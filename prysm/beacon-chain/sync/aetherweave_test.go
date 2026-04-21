package sync

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/iden3/go-iden3-crypto/v2/babyjub"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
	"github.com/iden3/go-rapidsnark/witness/v2"
	"github.com/iden3/go-rapidsnark/witness/wasmer"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/sync/awcontract"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestScoreDeterminism(t *testing.T) {
	pk1 := []byte("publickey1")
	pk2 := []byte("publickey2")
	nonce := Nonce(42)

	score1 := score(pk1, pk2, nonce)
	score2 := score(pk1, pk2, nonce)

	assert.Equal(t, score1, score2, "Score should be deterministic")
	assert.True(t, score1 >= 0.0 && score1 <= 1.0, "Score should be normalized")
}

func TestMerkleTreeAndProof(t *testing.T) {
	leaves := [][]byte{
		[]byte("leaf1"),
		[]byte("leaf2"),
		[]byte("leaf3"),
		[]byte("leaf4"),
	}

	tree, err := build_merkle_tree(leaves)
	require.NoError(t, err)
	require.True(t, len(tree) > 0, "Tree should have at least one level")

	proof, err := build_merkle_proof(2, tree) // proof for "leaf3"
	require.NoError(t, err)

	opening := &pb.CommitmentOpening{
		ParentHash: &pb.Hash{Hash: tree[len(tree)-1][0][:]},
		LeafIndex:  2,
		Proof:      make([]*pb.Hash, len(proof)),
	}
	for i, h := range proof {
		opening.Proof[i] = &pb.Hash{Hash: h[:]}
	}

	valid, err := verify_commitment_opening(opening, leaves[2])
	require.NoError(t, err)
	assert.True(t, valid, "Merkle proof should verify correctly")
}

// TestValidateSlashProof_Success ensures two conflicting but valid commitments
// from the same peer produce a SlashProof that validates successfully.
func TestValidateSlashProof_Success(t *testing.T) {
	// Generate mock Aetherweave keys (BabyJubJub)
	priv, pub, _, stakeID, err := awKeys()
	require.NoError(t, err)
	require.NotNil(t, priv)
	require.NotNil(t, pub)

	// Prepare fake execution engine (can be nil for this isolated test)
	// var execEngine execution.EngineCaller = nil

	// Resolve test ZK circuit paths
	wasmPath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_circ_bytes, err := os.ReadFile(wasmPath)
	if err != nil {
		log.WithError(err).WithField("fpath", wasmPath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	zkeyPath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_pkey_bytes, err := os.ReadFile(zkeyPath)
	if err != nil {
		log.WithError(err).WithField("fpath", zkeyPath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}

	// Get witness calculator
	sh_wcalc, err := witness.NewCalculator(
		sh_circ_bytes,
		witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
	)
	require.NoError(t, err)

	// Build two different commitment roots (distinct hashes)
	var root1, root2 Hash
	hash1 := sha256modQ([]byte("root-1"))
	hash2 := sha256modQ([]byte("root-2"))
	copy(root1[:], hash1[:])
	copy(root2[:], hash2[:])

	roundNumber := RoundNumber(1)

	// Calculate share for each commitment root
	share1, err := calculate_share(priv, roundNumber, root1)
	require.NoError(t, err)
	share2, err := calculate_share(priv, roundNumber, root2)
	require.NoError(t, err)

	// Build valid share ZK proofs for each
	shareProof1, err := build_share_proof(priv, stakeID, root1, share1, roundNumber, sh_wcalc, sh_pkey_bytes)
	require.NoError(t, err)
	require.NotEmpty(t, shareProof1)

	shareProof2, err := build_share_proof(priv, stakeID, root2, share2, roundNumber, sh_wcalc, sh_pkey_bytes)
	require.NoError(t, err)
	require.NotEmpty(t, shareProof2)

	// Construct two valid CommitmentRecords for same peer
	commitmentA := &pb.CommitmentRecord{
		RootHash:    &pb.Hash{Hash: root1[:]},
		RoundNumber: uint64(roundNumber),
		SlashShare:  share1.Bytes(),
		ShareProof:  shareProof1,
	}

	commitmentB := &pb.CommitmentRecord{
		RootHash:    &pb.Hash{Hash: root2[:]},
		RoundNumber: uint64(roundNumber),
		SlashShare:  share2.Bytes(),
		ShareProof:  shareProof2,
	}

	// Marshal public key bytes for slashee
	pubkeyBytes, err := crypto.MarshalPublicKey(pub)
	require.NoError(t, err)

	// Construct SlashProof (two conflicting commitments from same peer)
	slash := &pb.SlashProof{
		Slashee:  &pb.PublicKey{Pubkey: pubkeyBytes},
		Record_1: commitmentA,
		Record_2: commitmentB,
	}

	// Optionally log SlashProof structure for inspection
	jsonData, _ := json.MarshalIndent(slash, "", "  ")
	t.Logf("SlashProof:\n%s", string(jsonData))

	// Validate the SlashProof
	vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
	sh_vkey_bytes, err := os.ReadFile(vkeyFpath)
	if err != nil {
		log.WithError(err).WithField("fpath", vkeyFpath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	err = validateSlashProof(slash, sh_vkey_bytes)
	require.NoError(t, err, "expected SlashProof to validate successfully")
}

func TestBJJTestSignVerify(t *testing.T) {
	// Generate key
	privKey := babyjub.PrivateKey{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22}
	priv, pub, err := crypto.BJJKeyPairFromKey(&privKey)
	require.NoError(t, err)
	require.NotNil(t, pub)

	data := make([]byte, 512)

	sig, err := priv.Sign(data)
	require.NoError(t, err)

	ok, err := pub.Verify(data, sig)
	require.NoError(t, err)
	require.True(t, ok)

}

// TestBuildNetworkRecord_Success verifies that build_network_record correctly
// builds and signs a valid NetworkRecord. Also verifies that validateNetworkRecord
// properly validates the NetworkRecord.
func TestBuildNetworkRecord_Success(t *testing.T) {

	// Generate ECDSA keys
	eth_privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	eth_pubkey := &eth_privkey.PublicKey

	// Create mock engine interface
	client := &mockEthEng{
		netID:   new(big.Int).SetInt64(1000),
		nonce:   0,
		gas:     new(big.Int).SetInt64(1),
		balance: new(big.Int).SetInt64(1000),
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

	proof := &awcontract.SparseMerkleTreeProof{
		Root:     root_32,
		Siblings: [][32]byte{{0x00}, sibling_2_32, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}, {0x00}},
		Key:      key_32,
		Value:    value_32,
	}

	// Create mock contract caller and transactor objects
	contract_address := common.HexToAddress(CONTRACTS["AetherWeavePrivate"])
	contract := &mockAwContract{
		Tx: gethtypes.NewTx(&gethtypes.LegacyTx{
			To:       &contract_address,
			Nonce:    1,
			GasPrice: new(big.Int).SetInt64(100),
			Gas:      1000,
			Value:    new(big.Int).SetInt64(5000),
			Data:     make([]byte, 10),
		}),
	}
	contractCaller := &mockAwContractCaller{
		PoseidonResult: [32]byte{0},
		ProofResult:    *proof,
		RootResult:     root_32,
	}

	// Paths to circuits, prover, and verifier keys
	st_wasmFpath := filepath.Join("testdata", "zk", ST_CIRC)
	st_zkeyFpath := filepath.Join("testdata", "zk", ST_PKEY)
	st_vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	sh_wasmFpath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_zkeyFpath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)

	// Create new Aetherweave object
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

	// Write round number and smart contract root into Aetherweave object
	aw.round_number = RoundNumber(calculateRound())
	aw.updateSCRoots()

	// Override keys with those from TestZKProofOfStake
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
	aw.node_privkey = priv
	aw.node_pubkey = netPK_bytes
	aw.stakeSK = stakeSK
	aw.stakeID = stakeID

	// Create a proof of stake and get the merkle root
	proof_of_stake, merkle_root, err := aw.build_proof_of_stake()
	require.NoError(t, err)

	// Create a valid multiaddress
	maddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9000")
	require.NoError(t, err)
	addrs := []ma.Multiaddr{maddr}

	// Build the network record
	nr, err := build_network_record(aw.node_privkey, proof_of_stake, addrs, merkle_root)
	require.NoError(t, err)
	require.NotNil(t, nr)

	// Validate fields
	require.NotNil(t, nr.PublicKey)
	require.NotNil(t, nr.ProofOfStake)
	require.NotNil(t, nr.Multiaddr)
	require.NotNil(t, nr.Signature)

	require.Equal(t, proof_of_stake, nr.ProofOfStake)
	require.True(t, bytes.Equal(aw.node_pubkey[:], nr.PublicKey.Pubkey[:]))
	require.Equal(t, addrs[0].Bytes(), nr.Multiaddr.Multiaddr)

	// Call validateNetworkRecord
	vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	st_vkey_bytes, err := os.ReadFile(vkeyFpath)
	if err != nil {
		log.WithError(err).WithField("fpath", vkeyFpath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	ok, err = validateNetworkRecord(nr, aw.sc_roots, aw.round_number, st_vkey_bytes)
	require.NoError(t, err)
	assert.True(t, ok)

	// Optional: check timestamp is within a reasonable range
	now := time.Now().Unix()
	require.InDelta(t, float64(now), float64(nr.Timestamp), 2.0, "timestamp not within expected range")
}

func TestBuildCommitments_Success(t *testing.T) {

	// Generate ECDSA keys
	eth_privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	eth_pubkey := &eth_privkey.PublicKey

	// Create mock engine interface
	client := &mockEthEng{
		netID:   new(big.Int).SetInt64(1000),
		nonce:   0,
		gas:     new(big.Int).SetInt64(1),
		balance: new(big.Int).SetInt64(1000),
	}

	// Create mock contract caller and transactor objects
	contract_address := common.HexToAddress(CONTRACTS["AetherWeavePrivate"])
	contract := &mockAwContract{
		Tx: gethtypes.NewTx(&gethtypes.LegacyTx{
			To:       &contract_address,
			Nonce:    1,
			GasPrice: new(big.Int).SetInt64(100),
			Gas:      1000,
			Value:    new(big.Int).SetInt64(5000),
			Data:     make([]byte, 10),
		}),
	}
	contractCaller := &mockAwContractCaller{}

	// Create new Aetherweave object
	st_wasmFpath := filepath.Join("testdata", "zk", ST_CIRC)
	st_zkeyFpath := filepath.Join("testdata", "zk", ST_PKEY)
	st_vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	sh_wasmFpath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_zkeyFpath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
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

	// Insert dummy entries into the records table
	for i := 0; i < 2*AW_REQ_NUM; i++ {
		peerid := PublicKeyHash(fmt.Sprintf("%v", gethcrypto.Keccak256Hash([]byte{byte(i)})))
		aw.table.records[peerid] = &pb.PeerRecord{
			NetRecord: &pb.NetworkRecord{
				PublicKey: &pb.PublicKey{Pubkey: []byte{byte(i)}},
			},
		}
		aw.table.idx_pub[peerid] = true
	}

	// Sample random peerIDs of peers in our table
	log.Info("Sampling Aetherweave table")
	public_keys, _, err := aw.table.samplePublicKeys(uint(AW_REQ_NUM), true, map[PublicKeyHash]bool{})
	require.NoError(t, err)

	// Build CommitmentOpenings for sampled peers
	log.Info("Building commitments")
	wasmPath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_circ_bytes, err := os.ReadFile(wasmPath)
	if err != nil {
		log.WithError(err).WithField("fpath", wasmPath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	zkeyPath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_pkey_bytes, err := os.ReadFile(zkeyPath)
	if err != nil {
		log.WithError(err).WithField("fpath", zkeyPath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	// Get witness calculator
	sh_wcalc, err := witness.NewCalculator(
		sh_circ_bytes,
		witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
	)
	require.NoError(t, err)

	comm_record, comm_openings, err := build_commitments(public_keys, aw.round_number, aw.node_privkey, aw.stakeID, sh_wcalc, sh_pkey_bytes)
	require.NoError(t, err)
	require.NotNil(t, comm_record)
	require.NotNil(t, comm_openings)

	// Verify each commitment opening against the root hash
	for i, opening := range comm_openings {
		ok, err := verify_commitment_opening(opening, public_keys[i])
		require.NoError(t, err)
		assert.True(t, ok, "Merkle proof should validate for peer %d", i)
	}
}

func TestBuildSlashProof(t *testing.T) {
	// Generate ECDSA keys
	eth_privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	eth_pubkey := &eth_privkey.PublicKey

	// Create mock engine interface
	client := &mockEthEng{
		netID: new(big.Int).SetInt64(1000),
	}

	// Create mock contract caller and transactor objects
	contract := &mockAwContract{
		Tx: gethtypes.NewTx(&gethtypes.LegacyTx{}),
	}
	contractCaller := &mockAwContractCaller{}

	// Create new Aetherweave object
	st_wasmFpath := filepath.Join("testdata", "zk", ST_CIRC)
	st_zkeyFpath := filepath.Join("testdata", "zk", ST_PKEY)
	st_vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	sh_wasmFpath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_zkeyFpath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
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

	// Insert dummy entries into the records table
	for i := 0; i < 2*AW_REQ_NUM; i++ {
		peerid := PublicKeyHash(fmt.Sprintf("%v", gethcrypto.Keccak256Hash([]byte{byte(i)})))
		aw.table.records[peerid] = &pb.PeerRecord{
			NetRecord: &pb.NetworkRecord{
				PublicKey: &pb.PublicKey{Pubkey: []byte{byte(i)}},
			},
		}
		aw.table.idx_pub[peerid] = true
	}

	// Sample random peerIDs of peers in our table
	log.Info("Sampling Aetherweave table")
	public_keys_set1, _, err := aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
	require.NoError(t, err)
	public_keys_set2, _, err := aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
	require.NoError(t, err)
	// Make sure the sampled keys are not equal
	for bytes.Equal(public_keys_set1[0], public_keys_set2[0]) {
		public_keys_set2, _, err = aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
		require.NoError(t, err)
	}

	// Build CommitmentOpenings twice for the same round. This results in different
	log.Info("Building commitments")
	comm_record_1, comm_openings_set1, err := build_commitments(public_keys_set1, aw.round_number, aw.node_privkey, aw.stakeID, aw.sh_wc, aw.sh_pkey_bytes)
	require.NoError(t, err)
	require.NotNil(t, comm_record_1)
	require.NotNil(t, comm_openings_set1)
	comm_record_2, comm_openings_set2, err := build_commitments(public_keys_set2, aw.round_number, aw.node_privkey, aw.stakeID, aw.sh_wc, aw.sh_pkey_bytes)
	require.NoError(t, err)
	require.NotNil(t, comm_record_2)
	require.NotNil(t, comm_openings_set2)

	// Build Peer Record
	peerRecord := &pb.PeerRecord{
		NetRecord: &pb.NetworkRecord{
			PublicKey: &pb.PublicKey{Pubkey: aw.node_pubkey},
		},
		Commitments: []*pb.CommitmentRecord{comm_record_1, comm_record_2},
	}

	// Build SlashProof
	proof := aw.buildSlashProof(peerRecord)
	require.NotNil(t, proof)
	assert.True(t, bytes.Equal(aw.node_pubkey[:], proof.Slashee.Pubkey[:]))

	// Validate the slash proof
	vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
	sh_vkey_bytes, err := os.ReadFile(vkeyFpath)
	if err != nil {
		log.WithError(err).WithField("fpath", vkeyFpath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	err = validateSlashProof(proof, sh_vkey_bytes)
	assert.NoError(t, err)
}

func TestProcessSlashProofs_Valid(t *testing.T) {
	// Generate ECDSA keys
	eth_privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	eth_pubkey := &eth_privkey.PublicKey

	// Create mock engine interface
	client := &mockEthEng{
		netID: new(big.Int).SetInt64(1000),
	}

	// Create mock contract caller and transactor objects
	contract := &mockAwContract{
		Tx: gethtypes.NewTx(&gethtypes.LegacyTx{}),
	}
	contractCaller := &mockAwContractCaller{}

	// Create new Aetherweave object
	st_wasmFpath := filepath.Join("testdata", "zk", ST_CIRC)
	st_zkeyFpath := filepath.Join("testdata", "zk", ST_PKEY)
	st_vkeyFpath := filepath.Join("testdata", "zk", ST_VKEY)
	sh_wasmFpath := filepath.Join("testdata", "zk", SH_CIRC)
	sh_zkeyFpath := filepath.Join("testdata", "zk", SH_PKEY)
	sh_vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
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

	// Insert dummy entries into the records table
	for i := 0; i < 2*AW_REQ_NUM; i++ {
		peerid := PublicKeyHash(fmt.Sprintf("%v", gethcrypto.Keccak256Hash([]byte{byte(i)})))
		aw.table.records[peerid] = &pb.PeerRecord{
			NetRecord: &pb.NetworkRecord{
				PublicKey: &pb.PublicKey{Pubkey: []byte{byte(i)}},
			},
		}
		aw.table.idx_pub[peerid] = true
	}

	// Sample random peerIDs of peers in our table
	log.Info("Sampling Aetherweave table")
	public_keys_set1, _, err := aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
	require.NoError(t, err)
	public_keys_set2, _, err := aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
	require.NoError(t, err)
	// Make sure the sampled keys are not equal
	for bytes.Equal(public_keys_set1[0], public_keys_set2[0]) {
		public_keys_set2, _, err = aw.table.samplePublicKeys(1, true, map[PublicKeyHash]bool{})
		require.NoError(t, err)
	}

	// Build CommitmentOpenings twice for the same round. This results in different
	comm_record_1, comm_openings_set1, err := build_commitments(public_keys_set1, aw.round_number, aw.node_privkey, aw.stakeID, aw.sh_wc, aw.sh_pkey_bytes)
	require.NoError(t, err)
	require.NotNil(t, comm_record_1)
	require.NotNil(t, comm_openings_set1)
	comm_record_2, comm_openings_set2, err := build_commitments(public_keys_set2, aw.round_number, aw.node_privkey, aw.stakeID, aw.sh_wc, aw.sh_pkey_bytes)
	require.NoError(t, err)
	require.NotNil(t, comm_record_2)
	require.NotNil(t, comm_openings_set2)

	// Create a valid multiaddress
	maddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9000")
	require.NoError(t, err)

	// Build Peer Record
	networkRecord, err := build_network_record(aw.node_privkey, new(pb.ZKP), []ma.Multiaddr{maddr}, Hash{})
	require.NoError(t, err)
	peerRecord := &pb.PeerRecord{
		NetRecord:   networkRecord,
		Commitments: []*pb.CommitmentRecord{comm_record_1, comm_record_2},
	}

	// Build SlashProof
	proof := aw.buildSlashProof(peerRecord)
	require.NotNil(t, proof)
	assert.True(t, bytes.Equal(aw.node_pubkey[:], proof.Slashee.Pubkey[:]))

	// Process the slash proof
	vkeyFpath := filepath.Join("testdata", "zk", SH_VKEY)
	sh_vkey_bytes, err := os.ReadFile(vkeyFpath)
	if err != nil {
		log.WithError(err).WithField("fpath", vkeyFpath).Error("Failed to load ZK proof file")
		require.NoError(t, err)
	}
	aw.table.processSlashProofs([]*pb.SlashProof{proof}, sh_vkey_bytes)

	// Check that the slashee was blacklisted
	_, peerID, err := processMarshalledPubkey(peerRecord.NetRecord.PublicKey.Pubkey)
	require.NoError(t, err)

	entry, ok := aw.table.blacklist[PublicKeyHash(peerID)]
	require.True(t, ok, "Expected peer to be blacklisted")
	require.NotNil(t, entry.slash_proof)
	require.Equal(t, proof, entry.slash_proof)

}

func TestMaintainRecordsTable(t *testing.T) {
	totalRecords := TABLE_SIZE + 10 // More than TABLE_SIZE

	// Setup dummy node key (used in scoring)
	_, pub, err := crypto.GenerateBJJKeyPair(rand.Reader)
	require.NoError(t, err)
	nodePubKey, err := crypto.MarshalPublicKey(pub)
	require.NoError(t, err)

	// Initialize Aetherweave and RecordsTable
	table := &RecordsTable{
		idx_pub:   make(map[PublicKeyHash]bool),
		nonce_pub: 42,
		records:   make(map[PublicKeyHash]*pb.PeerRecord),
		blacklist: make(map[PublicKeyHash]BlacklistEntry),
		records_m: sync.RWMutex{},
	}

	aw := &Aetherweave{
		node_pubkey: nodePubKey,
		table:       table,
	}

	// Populate more than TABLE_SIZE records with random peer keys
	for i := 0; i < totalRecords; i++ {
		_, pubKey, err := crypto.GenerateBJJKeyPair(rand.Reader)
		require.NoError(t, err)
		pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
		require.NoError(t, err)
		_, peerID, err := processMarshalledPubkey(pubKeyBytes)
		require.NoError(t, err)

		netRec := &pb.NetworkRecord{
			PublicKey: &pb.PublicKey{Pubkey: pubKeyBytes},
		}
		rec := &pb.PeerRecord{
			NetRecord: netRec,
		}
		table.records[PublicKeyHash(peerID)] = rec
		table.idx_pub[PublicKeyHash(peerID)] = true
	}

	// Sanity check
	require.Greater(t, len(aw.table.records), TABLE_SIZE)

	// Maintain table
	aw.table.maintainRecordsTable(aw.node_pubkey, true)

	// Final size should match TABLE_SIZE
	assert.Equal(t, TABLE_SIZE, len(aw.table.records))

}

// [WIP] Test runs properly, but because of randomness and the scoring function threshold, fails
// We will need to refactor this code and its use of constants to do proper testing.
// func TestProcessPeerRecords(t *testing.T) {
// 	const (
// 		round       = 7
// 		nonce       = 42
// 		peers_count = 5
// 	)

// 	type Peer struct {
// 		priv crypto.PrivKey
// 		pub  crypto.PubKey
// 		pos  *pb.CommitmentOpening
// 		crs  []*pb.CommitmentRecord
// 		ops  []*pb.CommitmentOpening
// 		nr   *pb.NetworkRecord
// 	}

// 	// Generate host keypair
// 	h_priv, h_pub, err := crypto.GenerateBJJKeyPair(rand.Reader)
// 	require.NoError(t, err)
// 	h_pubBytes, err := crypto.MarshalPublicKey(h_pub)
// 	require.NoError(t, err)

// 	// Setup the Aetherweave instance
// 	table := &RecordsTable{
// 		nonce:     Nonce(nonce),
// 		records:   make(map[PublicKeyHash]*pb.PeerRecord),
// 		blacklist: make(map[PublicKeyHash]BlacklistEntry),
// 		records_m: sync.RWMutex{},
// 	}
// 	aw := &Aetherweave{
// 		node_pubkey:  h_pubBytes,
// 		node_privkey: h_priv,
// 		table:        table,
// 	}

// 	// Generate peer keypairs
// 	peers := make([]Peer, peers_count)
// 	pubkeys := make([][]byte, peers_count)
// 	for i := 0; i < peers_count; i++ {
// 		priv, pub, err := crypto.GenerateBJJKeyPair(rand.Reader)
// 		require.NoError(t, err)
// 		peers[i] = Peer{priv: priv, pub: pub}
// 		pubBytes, err := crypto.MarshalPublicKey(peers[i].pub)
// 		require.NoError(t, err)
// 		pubkeys[i] = pubBytes
// 	}

// 	// Build the smart contract root
// 	sc_merkle, err := build_merkle_tree(pubkeys)
// 	sc_root := Hash(sc_merkle[len(sc_merkle)-1][0])
// 	require.NoError(t, err)

// 	// Add sc_root to host
// 	sc_roots := make(map[Hash]RoundNumber)
// 	sc_roots[sc_root] = round
// 	aw.sc_roots = sc_roots

// 	// Build each peer's internal objects
// 	for i := 0; i < peers_count; i++ {
// 		// Proof of stake
// 		proof, err := build_merkle_proof(i, sc_merkle)
// 		require.NoError(t, err)
// 		pbProof := make([]*pb.Hash, len(proof))
// 		for j, h := range proof {
// 			pbProof[j] = &pb.Hash{Hash: h[:]}
// 		}
// 		peers[i].pos = &pb.CommitmentOpening{
// 			ParentHash: &pb.Hash{Hash: sc_root[:]},
// 			LeafIndex:  uint32(i),
// 			Proof:      pbProof,
// 		}
// 		// CommitmentRecord and CommitmentOpening with host included
// 		cr, ops, err := build_commitments([]PublicKey{h_pubBytes}, round, peers[i].priv)
// 		require.NoError(t, err)
// 		peers[i].crs = make([]*pb.CommitmentRecord, 0)
// 		peers[i].crs = append(peers[i].crs, cr)
// 		peers[i].ops = ops
// 		// NetworkRecord
// 		peer_ma, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9000")
// 		require.NoError(t, err)
// 		nr, err := build_network_record(peers[i].priv, peers[i].pos, []ma.Multiaddr{peer_ma})
// 		require.NoError(t, err)
// 		peers[i].nr = nr
// 	}

// 	// Peer 0 will have conflicting CommitmentRecords
// 	for i := 0; i < 3; i++ {
// 		root := sha256.Sum256([]byte(fmt.Sprintf("%v", i)))
// 		commitment := &pb.CommitmentRecord{
// 			RootHash:    &pb.Hash{Hash: root[:]},
// 			RoundNumber: round,
// 			Signature:   &pb.Signature{},
// 		}
// 		err := signAWMessage(commitment, peers[0].priv)
// 		require.NoError(t, err)
// 		peers[0].crs = append(peers[0].crs, commitment)
// 	}

// 	// Peer 1 will be blacklisted on the host
// 	aw.table.blacklist[PublicKeyHash(peers[1].nr.PublicKey.Pubkey)] = BlacklistEntry{}

// 	// Build final peer records
// 	peerRecords := make([]*pb.PeerRecord, peers_count)
// 	for i := 0; i < peers_count; i++ {
// 		peerRecords[i] = &pb.PeerRecord{
// 			NetRecord:   peers[i].nr,
// 			Commitments: peers[i].crs,
// 		}
// 	}

// 	// Call the function under test
// 	slashProofs := aw.processPeerRecords(peerRecords, Nonce(nonce), false)

// 	// Confirm that a slash proof was generated for peer 0
// 	require.Len(t, slashProofs, 1)
// 	require.NotNil(t, slashProofs[0])
// 	assert.Equal(t, peers[0].nr.PublicKey.Pubkey, slashProofs[0].Slashee.Pubkey)

// 	// Confirm that the peer 0 was not added to the table (slashable peer skipped)
// 	_, peerID, err := processMarshalledPubkey(peers[0].nr.PublicKey.Pubkey)
// 	require.NoError(t, err)
// 	_, exists := aw.table.records[PublicKeyHash(peerID)]
// 	assert.False(t, exists)
// }

func TestSignAndVerifyAWMessage(t *testing.T) {
	// Generate a new keypair
	priv, pub, err := crypto.GenerateBJJKeyPair(rand.Reader)
	require.NoError(t, err)

	pubBytes, err := crypto.MarshalPublicKey(pub)
	require.NoError(t, err)

	// Construct a minimal signable message: NetworkRecord
	nr := &pb.NetworkRecord{
		PublicKey:    &pb.PublicKey{Pubkey: pubBytes},
		ProofOfStake: new(pb.ZKP),
		MerkleRoot:   &pb.Hash{Hash: make([]byte, 32)},
		Multiaddr:    &pb.AWMultiAddr{Multiaddr: []byte("/ip4/127.0.0.1/tcp/9000")},
		Timestamp:    uint64(time.Now().Unix()),
		Signature:    &pb.Signature{}, // will be filled in
	}

	// Sign the message
	err = signAWMessage(nr, priv)
	require.NoError(t, err)
	require.NotNil(t, nr.Signature)
	len_c := false
	if len(nr.Signature.Signature) <= 71 {
		len_c = true
	}
	require.True(t, len_c)

	// Verify the signature
	ok, err := verifyAWMessage(nr, pub)
	require.NoError(t, err)
	assert.True(t, ok, "Signature verification should succeed")
}

// ===== Mock Types =====

// mockEthEng is a stub that satisfies execution.EngineCaller.
type mockEthEng struct {
	netID   *big.Int
	nonce   uint64
	gas     *big.Int
	balance *big.Int
}

func (m *mockEthEng) NetworkID(ctx context.Context) (*big.Int, error) {
	return m.netID, nil
}

func (m *mockEthEng) ChainID(ctx context.Context) (*big.Int, error) {
	return m.netID, nil
}

func (m *mockEthEng) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	m.nonce += 1
	return m.nonce, nil
}

func (m *mockEthEng) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return m.gas, nil
}

func (m *mockEthEng) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return new(big.Int).SetInt64(0), nil
}

func (m *mockEthEng) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return m.balance, nil
}

// mockAwContractCaller is a stub that satisfies the awContractCaller interface.
type mockAwContractCaller struct {
	PoseidonResult [32]byte
	ProofResult    awcontract.SparseMerkleTreeProof
	RootResult     [32]byte
}

// Poseidon1 simulates the contract call for the Poseidon hash function.
func (m *mockAwContractCaller) Poseidon1(opts *bind.CallOpts, el1_ [32]byte) ([32]byte, error) {
	return m.PoseidonResult, nil
}

// GetProof simulates the contract call to retrieve a Merkle proof.
func (m *mockAwContractCaller) GetProof(opts *bind.CallOpts, _stakeID *big.Int) (awcontract.SparseMerkleTreeProof, error) {
	return m.ProofResult, nil
}

// GetRoot simulates the contract call to retrieve the Merkle root.
func (m *mockAwContractCaller) GetRoot(opts *bind.CallOpts) ([32]byte, error) {
	return m.RootResult, nil
}

// mockAwContract is a stub that satisfies the awContract interface.
type mockAwContract struct {
	Tx *gethtypes.Transaction // The mock transaction to return
}

// Deposit simulates a contract transaction for depositing.
func (m *mockAwContract) Deposit(opts *bind.TransactOpts, _stakeID *big.Int) (*gethtypes.Transaction, error) {
	return m.Tx, nil
}

// Slash simulates a contract transaction for slashing.
func (m *mockAwContract) Slash(opts *bind.TransactOpts, _stakeSecret *big.Int, _stakeID *big.Int) (*gethtypes.Transaction, error) {
	return m.Tx, nil
}

// Init simulates a contract transaction for initializing the contract.
func (m *mockAwContract) Init(opts *bind.TransactOpts, _maxTreeDepth uint32, _stakeUnit *big.Int, _epochLength *big.Int, _withdrawalDelay *big.Int, _stakeFreezePeriod *big.Int) (*gethtypes.Transaction, error) {
	return m.Tx, nil
}
