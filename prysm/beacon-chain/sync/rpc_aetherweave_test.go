package sync

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	gorand "crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	mock "github.com/OffchainLabs/prysm/v6/beacon-chain/blockchain/testing"
	db "github.com/OffchainLabs/prysm/v6/beacon-chain/db/testing"
	p2ptest "github.com/OffchainLabs/prysm/v6/beacon-chain/p2p/testing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/startup"
	awc "github.com/OffchainLabs/prysm/v6/beacon-chain/sync/awcontract"
	"github.com/OffchainLabs/prysm/v6/config/params"
	leakybucket "github.com/OffchainLabs/prysm/v6/container/leaky-bucket"
	"github.com/OffchainLabs/prysm/v6/crypto/rand"
	pb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/testing/util"
	"github.com/iden3/go-iden3-crypto/v2/babyjub"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
	"github.com/iden3/go-rapidsnark/witness/v2"
	"github.com/iden3/go-rapidsnark/witness/wasmer"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p"
)

// Create two Sync Services with their own Aetherweave instances,
// send a request from one to the other, and process it on the receiving side
func TestAWRPCHandler(t *testing.T) {

	// Override network timeout parameters to avoid timeouts while debugging
	// 5 and 10 respectively are defaults for mainnet in production
	params.BeaconConfig().TtfbTimeout = 5
	params.BeaconConfig().RespTimeout = 10

	const (
		peers_count = 2
	)

	type Peer struct {
		aw     *Aetherweave
		peerID peer.ID
		pos    *pb.ZKP
		crs    []*pb.CommitmentRecord
		ops    []*pb.CommitmentOpening
		nr     *pb.NetworkRecord
		pr     *pb.PeerRecord
		ip     string
		ma     ma.Multiaddr
	}

	// Prepare a valid sample SparseMerkleTreeProof for the privKey used below
	var root_32 [32]byte
	root, ok := new(big.Int).SetString("19c7d89a2eaa512774a5540825b0d247c0402710dd4ba833cadeb0ccceff3335", 16)
	require.True(t, ok)
	root_b := root.Bytes()
	copy(root_32[32-len(root_b):], root_b)

	var key_32 [32]byte
	key, ok := new(big.Int).SetString("14cb82e52e164c80de23faef6ac666b90f20a922b4d350e245025cec4afdcf5a", 16)
	require.True(t, ok)
	key_b := key.Bytes()
	copy(key_32[32-len(key_b):], key_b)

	var sibling_2_32 [32]byte
	sibling_2, ok := new(big.Int).SetString("150cbdc21f8330ef67c1029e8d3950d75ca6c3d1788fe2084181688f19910c17", 16)
	sibling_2_b := sibling_2.Bytes()
	require.True(t, ok)
	copy(sibling_2_32[32-len(sibling_2_b):], sibling_2_b)

	var value_32 [32]byte
	value, ok := new(big.Int).SetString("0000000000000000000000000000000000000000000000000000000000000001", 16)
	value_b := value.Bytes()
	require.True(t, ok)
	copy(value_32[32-len(value_b):], value_b)

	proof := &awc.SparseMerkleTreeProof{
		Root:     root_32,
		Siblings: [][32]byte{{0x00}, sibling_2_32, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}},
		Key:      key_32,
		Value:    value_32,
	}

	// Create mock engine interface
	client := &mockEthEng{
		netID: new(big.Int).SetInt64(1000),
	}

	// Create mock contract caller and transactor objects
	contract := &mockAwContract{
		Tx: gethtypes.NewTx(&gethtypes.LegacyTx{}),
	}
	contractCaller := &mockAwContractCaller{
		ProofResult: *proof,
		RootResult:  root_32,
	}

	// ZK Proof files
	st_wasmFpath := filepath.Join("testdata", "zk", ST_CIRC)
	st_zkeyFpath := filepath.Join("testdata", "zk", ST_PKEY)
	st_vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	sh_wasmFpath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_zkeyFpath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)

	// Generate key pairs
	peers := make([]Peer, peers_count)
	for i := 0; i < peers_count; i++ {
		// Generate ECDSA keys
		eth_privkey, err := ecdsa.GenerateKey(elliptic.P256(), gorand.Reader)
		require.NoError(t, err)
		eth_pubkey := &eth_privkey.PublicKey
		// Create Aetherweave object for peer
		aw, err := NewAetherweave(
			client,
			contract,
			contractCaller,
			eth_privkey,
			eth_pubkey,
			st_wasmFpath,
			st_zkeyFpath,
			st_vkeyFpath,
			sh_wasmFpath,
			sh_zkeyFpath,
			sh_vkeyFpath,
		)
		require.NoError(t, err)
		peers[i].aw = aw
		// Write round number and smart contract root into Aetherweave object
		aw.round_number = RoundNumber(calculateRound())
		aw.updateSCRoots()
		// Generate peerID from public key
		peerID, err := peer.IDFromPublicKey(peers[i].aw.node_privkey.GetPublic())
		assert.NoError(t, err)
		peers[i].peerID = peerID
	}

	// Overwrite p1 keys with the ones used to generate the above valid proof of stake. Sourced from TestZKProofOfStake
	privKey := babyjub.PrivateKey{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22}
	priv, pub, err := crypto.BJJKeyPairFromKey(&privKey)
	require.NoError(t, err)
	require.NotNil(t, pub)
	netPK_bytes, err := crypto.MarshalPublicKey(pub)
	require.NoError(t, err)
	bjj_privkey, ok := priv.(*crypto.BJJPrivateKey)
	require.True(t, ok)
	stakeSK, err := poseidon.Hash([]*big.Int{bjj_privkey.SkToBigInt()})
	require.NoError(t, err)
	stakeID, err := poseidon.Hash([]*big.Int{stakeSK})
	require.NoError(t, err)
	peers[0].aw.node_privkey = priv
	peers[0].aw.node_pubkey = netPK_bytes
	peers[0].aw.stakeSK = stakeSK
	peers[0].aw.stakeID = stakeID

	// Options to use the generated keys with the mock p2p interfaces
	p1_privkey_opt := privKeyOption(peers[0].aw.node_privkey)
	p2_privkey_opt := privKeyOption(peers[1].aw.node_privkey)

	// Setup new P2P services
	p1 := p2ptest.NewTestP2P(t, p1_privkey_opt)
	p2 := p2ptest.NewTestP2P(t, p2_privkey_opt)

	// Connect p1 to p2
	p1.Connect(p2)

	// Verify that hosts have been connected
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	// Set up a head state in the database with data we expect.
	d := db.SetupDB(t)

	// Load the wasm and zkey files for stake and share proofs
	st_circ_path := filepath.Join("testdata/zk", ST_CIRC)
	st_pkey_path := filepath.Join("testdata/zk", ST_PKEY)
	st_vkey_path := filepath.Join("testdata/zk", ST_VKEY)
	sh_circ_path := filepath.Join("testdata/zk", SH_CIRC)
	sh_pkey_path := filepath.Join("testdata/zk", SH_PKEY)
	sh_vkey_path := filepath.Join("testdata/zk", SH_VKEY)

	// Load stake and share proof files
	files_bytes := make([][]byte, 6)
	for idx, fpath := range []string{st_circ_path, st_pkey_path, st_vkey_path, sh_circ_path, sh_pkey_path, sh_vkey_path} {
		file_bytes, err := os.ReadFile(fpath)
		require.NoError(t, err)
		files_bytes[idx] = file_bytes
	}

	// Overwrite proof bytes
	for _, i := range []int{0, 1} {
		peers[i].aw.st_circ_bytes = files_bytes[0]
		peers[i].aw.st_pkey_bytes = files_bytes[1]
		peers[i].aw.st_vkey_bytes = files_bytes[2]
		peers[i].aw.sh_circ_bytes = files_bytes[3]
		peers[i].aw.sh_pkey_bytes = files_bytes[4]
		peers[i].aw.sh_vkey_bytes = files_bytes[5]
	}

	wcalc, err := witness.NewCalculator(
		peers[0].aw.sh_circ_bytes,
		witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
	)
	require.NoError(t, err)

	// Prepare p1 heartbeat: pick nonce, build proof of stake, get multiaddr, build network record, build commitment opening
	peers[0].aw.table.nonce_pub = Nonce(rand.NewGenerator().Uint64())
	p1_proof_of_stake, merkle_root, err := peers[0].aw.build_proof_of_stake()
	peers[0].pos = p1_proof_of_stake
	require.NoError(t, err)
	// p1_ma, err := p1.GetMultiAddrs()
	// peers[0].ma = p1_ma[0]
	// require.NoError(t, err)
	peers[0].ip = fmt.Sprintf("/ip4/%s/tcp/%d", "127.0.0.1", 5001)
	peer_ma, err := ma.NewMultiaddr(peers[0].ip)
	peers[0].ma = peer_ma
	require.NoError(t, err)
	node_net_record, err := build_network_record(peers[0].aw.node_privkey, peers[0].pos, []ma.Multiaddr{peer_ma}, merkle_root)
	peers[0].nr = node_net_record
	require.NoError(t, err)
	comm_record, comm_openings, err := build_commitments([]PublicKey{peers[1].aw.node_pubkey}, peers[0].aw.round_number, peers[0].aw.node_privkey, peers[0].aw.stakeID, wcalc, peers[0].aw.sh_pkey_bytes)
	peers[0].crs = []*pb.CommitmentRecord{comm_record}
	peers[0].ops = comm_openings
	require.NoError(t, err)

	// Build p1's request to p2
	p1_request := &pb.Request{
		Nonces:            []uint64{uint64(peers[0].aw.table.nonce_pub)},
		SenderRecord:      node_net_record,
		CommitmentRecord:  comm_record,
		CommitmentOpening: peers[0].ops[0],
		Signature:         &pb.Signature{},
	}
	err = signAWMessage(p1_request, peers[0].aw.node_privkey)
	require.NoError(t, err)

	// Inject synthetic peer records into p2 to respond to p1's request
	for i := 0; i < 100; i++ {
		p_ma, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", "127.0.0.1", 6000+i))
		require.NoError(t, err)
		p_ma_b, err := p_ma.MarshalBinary()
		require.NoError(t, err)
		p_pub := make([]byte, 36)
		p_pub[0] = uint8(i)
		pr := &pb.PeerRecord{
			NetRecord: &pb.NetworkRecord{
				PublicKey:    &pb.PublicKey{Pubkey: p_pub},
				ProofOfStake: new(pb.ZKP),
				MerkleRoot:   &pb.Hash{Hash: merkle_root[:]},
				Multiaddr:    &pb.AWMultiAddr{Multiaddr: p_ma_b},
				Timestamp:    uint64(time.Now().Unix()),
				Signature:    &pb.Signature{},
			},
			Commitments: []*pb.CommitmentRecord{
				&pb.CommitmentRecord{
					RootHash:    &pb.Hash{Hash: make([]byte, 32)},
					RoundNumber: uint64(peers[0].aw.round_number),
					SlashShare:  make([]byte, 32),
					ShareProof:  new(pb.ZKP),
				},
			},
		}
		peers[1].aw.table.records[PublicKeyHash(fmt.Sprintf("%v", i))] = pr
		peers[1].aw.table.idx_pub[PublicKeyHash(fmt.Sprintf("%v", i))] = true
	}

	// Setup Chain and Sync services for p1
	chain := &mock.ChainService{ValidatorsRoot: [32]byte{}, Genesis: time.Now()}
	r1 := &Service{
		aw: peers[0].aw,
		cfg: &config{
			beaconDB: d,
			p2p:      p1,
			chain:    chain,
			clock:    startup.NewClock(chain.Genesis, chain.ValidatorsRoot),
		},
		rateLimiter: newRateLimiter(p1),
	}

	// Add records of each peer to the other
	p1.Peers().Add(new(enr.Record), p2.BHost.ID(), p2.BHost.Addrs()[0], network.DirUnknown)
	p2.Peers().Add(new(enr.Record), p1.BHost.ID(), p1.BHost.Addrs()[0], network.DirUnknown)

	// Setup Chain and Sync services for p2
	chain2 := &mock.ChainService{ValidatorsRoot: [32]byte{}, Genesis: time.Now()}
	r2 := &Service{
		aw: peers[1].aw,
		cfg: &config{
			beaconDB: d,
			p2p:      p2,
			chain:    chain2,
			clock:    startup.NewClock(chain2.Genesis, chain.ValidatorsRoot),
		},
		rateLimiter: newRateLimiter(p2),
	}

	// Build request from r1 to r2
	request := &pb.Request{
		Nonces:            []uint64{uint64(peers[0].aw.table.nonce_pub)},
		SenderRecord:      peers[0].nr,
		CommitmentRecord:  peers[0].crs[0],
		CommitmentOpening: peers[0].ops[0],
	}

	// Setup streams
	pcl := protocol.ID("/aetherweave/heartbeat/1/ssz_snappy")
	topic := string(pcl)
	r2.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, time.Second, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Request{}
		// Decode message
		assert.NoError(t, r2.cfg.p2p.Encoding().DecodeWithMaxLength(stream, out))
		assert.Equal(t, uint64(peers[0].aw.round_number), out.GetCommitmentRecord().GetRoundNumber())
		assert.NoError(t, r2.aw_requestRPCHandler(context.Background(), out, stream))
	})

	// Send request
	assert.NoError(t,
		r1.sendRPCAWRequest(context.Background(), request, p2.BHost.ID(), []Nonce{peers[0].aw.table.nonce_pub}))

	if util.WaitTimeout(&wg, 5*time.Second) {
		t.Fatal("Did not receive stream within 5 sec")
	}

	conns := p1.BHost.Network().ConnsToPeer(p2.BHost.ID())
	if len(conns) == 0 {
		t.Error("Peer is disconnected despite receiving a valid ping")
	}
}

// Adds a private key to the libp2p option if the option was provided.
// Built off the function with the same name in p2p/options.go
func privKeyOption(privkey crypto.PrivKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		return cfg.Apply(libp2p.Identity(privkey))
	}
}
