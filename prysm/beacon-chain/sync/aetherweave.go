package sync

import (
	"bytes"
	"context"
	gorand "crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/sync/awcontract"
	"github.com/OffchainLabs/prysm/v6/cmd"
	"github.com/OffchainLabs/prysm/v6/cmd/beacon-chain/flags"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/io/file"
	"github.com/OffchainLabs/prysm/v6/monitoring/tracing/trace"
	pb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/OffchainLabs/prysm/v6/crypto/rand"

	"github.com/spaolacci/murmur3"

	"crypto/sha256"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	snarkconst "github.com/iden3/go-iden3-crypto/v2/constants"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
	snarktypes "github.com/iden3/go-rapidsnark/types"
	"github.com/iden3/go-rapidsnark/witness/v2"
	"github.com/iden3/go-rapidsnark/witness/wasmer"

	"crypto/ecdsa"
	"crypto/elliptic"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
)

// Aetherweave Parameters

// Aetherweave round start time in epoch seconds
const AW_GENESIS int64 = 1753747200

// Round time in seconds
const AW_ROUND_TIME int64 = 2 * 60

// Round cutoff time for running heartbeat
const AW_CUTOFF_SPAN int64 = int64(0.9 * float64(AW_ROUND_TIME))

// Ceil(Square root(Number of nodes in the protocol))
var AW_NODES_SQ = 10

// Number of nodes participating in the protocol
var AW_NODES_NUM = AW_NODES_SQ * AW_NODES_SQ

// Scaling factor
const AW_SCALE = 4

// Maximum number of peers to keep in our records table
var TABLE_SIZE = AW_SCALE * AW_NODES_SQ

// Number of peers to which we make requests every round
var AW_REQ_NUM = TABLE_SIZE

// Time for which a node will remember slashings that occurred in the network in seconds
const BL_TIME = 14 * 24 * 60 * 60

// Maximum number of SlashProofs in a response
const MAX_RES_SLASHPROOFS = 100

// Maximum number of rounds ago for which to consider smart contract roots valid
const MAX_SC_ROOT_AGE = 3

// Number of recent Commitments to keep per peer
const COMMITMENT_HIST_SIZE = 3

// Signature buffer size
const SIG_SIZE = 71

// Share proof constant. Used for generating shares
const SP_CONST = 448612363379

// Contract initialization parameters
// Amount of stake in Wei that each participant in the protocol needs to stake in the contract
const STAKE_UNIT = 1e18

// Merkle tree depth
const TREE_DEPTH = 32

// Epoch time in seconds
const AW_EPOCH_TIME = 10 * 24 * 60

// Stake freeze period in seconds
const ST_FREEZE_T = 30

// Stake withdrawal time after request in seconds
const ST_WITHDRAW_T = 30

// StakeProof caching. We only need to regenerate stake proofs when the stake root changes.
const ST_CACHE = true

// Cache NetworkRecords and skip checks for records we've already seen
const NETREC_CACHE = true

// Enable CPU profiling
const CPU_PROFILING = false

// Aetherweave smart contracts addresses
var CONTRACTS = map[string]string{
	"SparseMerkleTree":   "0x1111B44847b379578588920cA78fbf26C0b4956C",
	"PoseidonT2":         "0x2222B44847B379578588920Ca78Fbf26C0b4956c",
	"PoseidonT3":         "0x3333B44847b379578588920cA78fbf26C0B4956C",
	"PoseidonT4":         "0x4444B44847b379578588920Ca78FBf26c0B4956c",
	"AetherWeavePrivate": "0x5555B44847b379578588920Ca78FBf26c0B4956c",
}

// Dual table support: If enabled, we keep two internal peer tables,
// one for forming connections, and another for responding to queries
const DUAL_TABLES = false

type Nonce uint64
type Score float64

type Timestamp uint64
type RoundNumber uint64
type Hash [32]byte
type PrivateKey crypto.PrivKey
type PublicKey []byte
type PrivateKeyHash string
type PublicKeyHash peer.ID
type Signature [SIG_SIZE]byte

// Interface for *ethclient.Client. We abstract it for testing purposes
type ethEng interface {
	NetworkID(ctx context.Context) (*big.Int, error)
	ChainID(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

// Interface for the Aetherweave contract caller. We abstract it for testing purposes
type awContractCaller interface {
	Poseidon1(opts *bind.CallOpts, el1_ [32]byte) ([32]byte, error)
	GetProof(opts *bind.CallOpts, _stakeID *big.Int) (awcontract.SparseMerkleTreeProof, error)
	GetRoot(opts *bind.CallOpts) ([32]byte, error)
}

// Interface for the Aetherweave contract. We abstract it for testing purposes
type awContract interface {
	Deposit(opts *bind.TransactOpts, _stakeID *big.Int) (*gethtypes.Transaction, error)
	Init(opts *bind.TransactOpts, _maxTreeDepth uint32, _stakeUnit *big.Int, _epochLength *big.Int, _withdrawalDelay *big.Int, _stakeFreezePeriod *big.Int) (*gethtypes.Transaction, error)
	Slash(opts *bind.TransactOpts, _stakeSecret *big.Int, _stakeID *big.Int) (*gethtypes.Transaction, error)
}

// Aetherweave state object
type Aetherweave struct {
	table           *RecordsTable
	node_pubkey     PublicKey
	node_privkey    PrivateKey
	eth_pubkey      *ecdsa.PublicKey
	eth_privkey     *ecdsa.PrivateKey // Aetherweave ECDSA private key for interacting with smart contract
	stakeSK         *big.Int
	stakeID         *big.Int
	round_number    RoundNumber
	sc_roots        *SCRootsTable
	ethClient       ethEng
	contract        awContract
	contractCaller  awContractCaller
	net_record_path string
	net_records_dir string
	sh_vkey_bytes   []byte
	sh_pkey_bytes   []byte
	sh_circ_bytes   []byte
	sh_wc           witness.Calculator
	st_vkey_bytes   []byte
	st_pkey_bytes   []byte
	st_circ_bytes   []byte
	st_wc           witness.Calculator

	st_cache_zkp    *pb.ZKP
	st_cache_merkle Hash
}

// Table to hold recent smart contract merkle roots
type SCRootsTable struct {
	sc_table   map[Hash]RoundNumber
	sc_table_m sync.RWMutex
}

// Aetherweave record table object
type RecordsTable struct {
	idx_pub      map[PublicKeyHash]bool           // Indexes for the public table
	idx_priv     map[PublicKeyHash]bool           // Indexes for the private table
	nonce_pub    Nonce                            // Nonce for the public table
	nonce_priv   Nonce                            // Nonde for the private table
	records      map[PublicKeyHash]*pb.PeerRecord // Table with all records
	records_m    sync.RWMutex
	blacklist    map[PublicKeyHash]BlacklistEntry
	blacklist_m  sync.RWMutex
	served_com   map[Hash]RoundNumber
	served_com_m sync.RWMutex
}

type BlacklistEntry struct {
	slash_proof *pb.SlashProof
	timestamp   time.Time
}

type Signable interface {
	MarshalSSZ() ([]byte, error)
	GetSignature() *pb.Signature
}

// Calculates the current round number
func calculateRound() uint64 {
	current_ts := time.Now().Unix()
	return uint64((current_ts - AW_GENESIS) / AW_ROUND_TIME)
}

// Returns the epoch start time of the given round number
func calculateRoundStartTime(round_n int64) int64 {
	return AW_GENESIS + (round_n * AW_ROUND_TIME)
}

// Checks if Aetherweave contracts have been deployed
func checkAWContracts(client *ethclient.Client) {
	log.Info("Checking Aetherweave contracts")
	for contract, address := range CONTRACTS {
		sc := common.HexToAddress(address)
		code, err := client.CodeAt(context.Background(), sc, nil)
		if err != nil || len(code) == 0 {
			log.WithError(err).WithFields(logrus.Fields{"contract": contract, "address": address}).Error("Failed to fetch contract")
		} else {
			log.WithFields(logrus.Fields{"contract": contract, "address": address}).Info("Found contract")
		}
	}
}

// Function to test interaction with chain and check wallet and smart contract balances, and also contract interactions
func queryChain(client ethEng, contract awContractCaller, eth_pubkey *ecdsa.PublicKey) {

	// Initialize clients
	// log.Info("Aetherweave: running queryChain")

	// Check account balance
	account := gethcrypto.PubkeyToAddress(*eth_pubkey)
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		log.WithError(err).Error("Failed to fetch account balance")
	}
	log.WithField("balance", balance).Info("Staker wallet balance")

	// Check contracts
	// for contract, address := range CONTRACTS {
	// 	sc := common.HexToAddress(address)
	// 	sc_balance, err := client.BalanceAt(context.Background(), sc, nil)
	// 	if err != nil {
	// 		log.WithError(err).Error("Failed to fetch account balance")
	// 	}
	// 	log.WithFields(logrus.Fields{"balance": sc_balance, "contract": contract}).Info("Got contract balance")
	// }

	// Only check AetherWeave contract balance
	{
		contract := "AetherWeavePrivate"
		address := CONTRACTS[contract]
		sc := common.HexToAddress(address)
		sc_balance, err := client.BalanceAt(context.Background(), sc, nil)
		if err != nil {
			log.WithError(err).Error("Failed to fetch account balance")
		}
		log.WithFields(logrus.Fields{"balance": sc_balance, "contract": contract}).Info("Got contract balance")
	}

	// Test interacting with contract
	// Input to Poseidon1
	input := [32]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}

	// Call the contract function
	opts := &bind.CallOpts{
		Context: context.Background(),
	}
	result, err := contract.Poseidon1(opts, input)
	if err != nil {
		log.WithError(err).WithField("result", result).Error("Failed to call the Poseidon1 function")
	}
	// log.WithField("result", result).Info("Poseidon1 output")

}

// Sample public keys from the records table using the public or private table index
func (r *RecordsTable) samplePublicKeys(count uint, usePublicTable bool, excludeList map[PublicKeyHash]bool) ([]PublicKey, []PublicKeyHash, error) {

	r.records_m.RLock()
	defer r.records_m.RUnlock()

	// If dual tables is disabled and we want to use the private table, this is a logical error
	if !DUAL_TABLES && !usePublicTable {
		return nil, nil, errors.New("Cannot use private table with DUAL_TABLES disabled")
	}

	// Get the table index of the correct table
	var table_idx *map[PublicKeyHash]bool
	if usePublicTable {
		table_idx = &r.idx_pub
	} else {
		table_idx = &r.idx_priv
	}

	// Check if we have entries in the public / private table index
	if len(*table_idx) == 0 {
		return nil, nil, errors.New("table index is empty")
	}

	// This should never be true so long as we have entries in the table index,
	// but it is kept here as a precaution
	if len(r.records) == 0 {
		return nil, nil, errors.New("records table is empty")
	}

	// Build a list of all peerIDs
	keys := make([]PublicKeyHash, 0, len(*table_idx))
	for pk := range *table_idx {
		if _, exclude := excludeList[pk]; !exclude {
			// Check that the peerID also exists in the table. This should never fail
			if _, ok := r.records[pk]; !ok {
				log.WithFields(logrus.Fields{"aw_peerID": pk}).Warn("aw_peerID in index does not exist in records table. Skipping selection in samplePublicKeys()")
				continue
			}
			keys = append(keys, pk)
		}
	}

	// Shuffle peerIDs
	rng := rand.NewGenerator()
	rng.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	// Pick up to count peerIDs
	sampleSize := count
	if sampleSize > uint(len(keys)) {
		sampleSize = uint(len(keys))
	}
	selectedKeys := make([]PublicKeyHash, 0, sampleSize)
	publicKeys := make([]PublicKey, 0, sampleSize)

	for _, key := range keys[:sampleSize] {
		record := r.records[key]
		if record == nil || record.NetRecord == nil || record.NetRecord.PublicKey == nil {
			continue
		}
		publicKeys = append(publicKeys, record.NetRecord.PublicKey.Pubkey)
		selectedKeys = append(selectedKeys, key)
	}

	return publicKeys, selectedKeys, nil
}

func (r *RecordsTable) getRecord(peer PublicKeyHash) (*pb.PeerRecord, bool) {
	r.records_m.RLock()
	defer r.records_m.RUnlock()

	record, ok := r.records[peer]
	return record, ok
}

// Returns true if we've seen a SlashProof for a peer
func (r *RecordsTable) isPeerBlacklisted(peerID PublicKeyHash) bool {
	r.blacklist_m.RLock()
	defer r.blacklist_m.RUnlock()
	_, exists := r.blacklist[peerID]
	return exists
}

// Process a list of SlashProofs and update the Blacklist in RecordsTable
func (r *RecordsTable) processSlashProofs(slashProofs []*pb.SlashProof, sh_vkey_bytes []byte) {

	// Iterate over SlashProofs
	for _, new_slash := range slashProofs {

		// Validate SlashProof
		if err := validateSlashProof(new_slash, sh_vkey_bytes); err != nil {
			log.WithError(err).Error("Failed to validate SlashProof")
			continue
		}

		// Extract public key and calculate peerID
		_, slashee_peerID, err := processMarshalledPubkey(new_slash.GetSlashee().GetPubkey())
		if err != nil {
			log.WithError(err).Error("Failed to get peerID from public key")
			continue
		}

		// Check peer is already in the blacklist. If not, add it
		r.blacklist_m.RLock()
		_, ok := r.blacklist[PublicKeyHash(slashee_peerID)]
		r.blacklist_m.RUnlock()
		if !ok {
			r.blacklist_m.Lock()
			r.blacklist[PublicKeyHash(slashee_peerID)] = BlacklistEntry{
				slash_proof: new_slash,
				timestamp:   time.Now(),
			}
			r.blacklist_m.Unlock()
		}
	}
}

// Mark a Commitment as served. This is used when we get a Request and send a Response for the Commitment.
func (r *RecordsTable) markCommitmentServed(slash_share Hash, round_n RoundNumber) {
	r.served_com_m.Lock()
	defer r.served_com_m.Unlock()
	r.served_com[slash_share] = round_n
}

// Check if a Commitment has been previously served
func (r *RecordsTable) checkCommitmentServed(slash_share Hash) bool {
	r.served_com_m.RLock()
	defer r.served_com_m.RUnlock()
	_, ok := r.served_com[slash_share]
	return ok
}

// Prune old entries in our table of served commitments
func (r *RecordsTable) maintainServedCommitmentsTable(current_round RoundNumber) int {
	r.served_com_m.Lock()
	defer r.served_com_m.Unlock()
	keys := make([]Hash, 0)
	for key, round := range r.served_com {
		if round < current_round-2 {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		delete(r.served_com, key)
	}

	return len(keys)
}

// If the table is over the size limit, score all records, and keep highest scoring ones
func (rt *RecordsTable) maintainRecordsTable(node_pubkey PublicKey, usePublicTable bool) (int, int) {

	if !DUAL_TABLES && !usePublicTable {
		log.WithField("func", "maintainRecordsTable").Error("Should not use private table with DUAL_TABLES disabled")
		return 0, 0
	}

	// Get the table index that we will be updating
	idx := &rt.idx_pub
	other_idx := &rt.idx_priv
	nonce := rt.nonce_pub
	if !usePublicTable {
		idx = &rt.idx_priv
		nonce = rt.nonce_priv
		other_idx = &rt.idx_pub
	}

	// Get lock on records table
	rt.records_m.Lock()
	defer rt.records_m.Unlock()

	// If table index is not over the size limit, no need to maintain it
	current_table_size := len(*idx)
	if current_table_size <= TABLE_SIZE {
		return 0, 0
	}

	// Find the score cutoff point
	all_scores := make([]Score, 0, current_table_size)
	keys_scores := make(map[PublicKeyHash]Score, current_table_size)
	for key := range *idx {
		record := rt.records[key]
		record_score := score(node_pubkey, record.GetNetRecord().GetPublicKey().GetPubkey(), nonce)
		all_scores = append(all_scores, record_score)
		keys_scores[key] = record_score
	}
	// Sort scores by ascending order
	sort.Slice(all_scores, func(i, j int) bool { return all_scores[i] < all_scores[j] })
	cutoff := all_scores[TABLE_SIZE-1]

	// Get keys that will be deleted
	del_idx_counter := 0
	del_table_counter := 0
	for key, score := range keys_scores {
		if score > cutoff {
			// Delete key from index
			delete(*idx, key)
			del_idx_counter += 1
			// If the key is not used in the other index, also delete it from the table
			if ok, _ := (*other_idx)[key]; !ok {
				delete(rt.records, key)
				del_table_counter += 1
			}
		}
	}
	return del_idx_counter, del_table_counter
}

// Process a list of PeerRecords to update the RecordsTable. Return SlashProofs for identified Commitment collisions, and a flag indicating if any record failed validation
func (aw *Aetherweave) processPeerRecord(new_record *pb.PeerRecord, requestNonce Nonce, pubkey crypto.PubKey, aw_peerID peer.ID, inject bool, usePublicTable bool) (*pb.SlashProof, error) {

	// Get the table index that we will be updating
	idx := &aw.table.idx_pub
	if !usePublicTable {
		idx = &aw.table.idx_priv
	}

	// Get NetRecord
	net_record := new_record.GetNetRecord()

	// Check if the record is already in our table
	valid_rec_exists := false
	if NETREC_CACHE {
		aw.table.records_m.RLock()
		record, record_exists := aw.table.records[PublicKeyHash(aw_peerID)]
		aw.table.records_m.RUnlock()

		// If the merkle roots of the two records match, we don't need to verify the proof of stake again,
		// just authenticate the record in case it has been updated.
		if record_exists && bytes.Equal(record.GetNetRecord().GetMerkleRoot().GetHash(), net_record.GetMerkleRoot().GetHash()) {
			valid_rec_exists = true

			// Verify NetworkRecord signature
			ok, err := verifyAWMessage(net_record, pubkey)
			if err != nil {
				log.WithError(err).Error("Failed to authenticate new record for entry already in tables")
				return nil, err
			}
			if !ok {
				log.Error("invalid NetworkRecord signature for record with entry already in tables")
				return nil, err
			}
		}
	}

	// Check NetworkRecord multiaddr validity
	record_ma, err := ma.NewMultiaddrBytes(new_record.GetNetRecord().GetMultiaddr().GetMultiaddr())
	if err != nil {
		log.WithFields(logrus.Fields{"function": "processPeerRecord", "aw_peerID": aw_peerID}).WithError(err).Error("Could not unmarshall peer addrinfo. Not inserting PeerRecord.")
		return nil, err
	}

	// Get the native peerID from the multiaddr
	peerID, err := GetPeerIDFromMultiaddr(record_ma)
	if err != nil {
		log.WithField("peerID", peerID).Error("Failed to GetPeerIDFromMultiaddr() when processing PeerRecord")
		return nil, err
	}

	// Skip processing peer if we have a SlashProof for it
	if aw.table.isPeerBlacklisted(PublicKeyHash(aw_peerID)) {
		log.Warnf("Discarding blacklisted peer record %v", new_record)
		return nil, err
	}

	// If we have never seen this record before, calculate its score for the table, and validate it
	if !valid_rec_exists {
		// Calculate score for each record, and skip processing it if it doesn't meet the scoring criterion
		// If record function is called for injecting a record, skip the score test
		rec_score := score(aw.node_pubkey, new_record.NetRecord.PublicKey.GetPubkey(), requestNonce)
		if !inject && float64(rec_score) > float64(TABLE_SIZE)/float64(AW_NODES_NUM) {
			log.Warnf("Received peer record over the scoring criterion %v", new_record)
			return nil, err
		}

		// Validate the network record
		ok, err := validateNetworkRecord(net_record, aw.sc_roots, aw.round_number, aw.st_vkey_bytes)
		if err != nil {
			log.WithError(err).Error("Received NetworkRecord validation failed")
			return nil, err
		}
		if !ok {
			return nil, err
		}
	}

	// Merge old record with new one
	aw.table.records_m.Lock()
	if old_record, ok := aw.table.records[PublicKeyHash(aw_peerID)]; ok {
		// Build new network record commitments
		new_record.Commitments = append(new_record.Commitments, old_record.Commitments...)

		// Check for freshness of NetworkRecord
		if old_record.NetRecord.Timestamp > new_record.NetRecord.Timestamp {
			new_record.NetRecord = old_record.NetRecord
		}
	}

	// Limit number of commitments we hold for this peer
	if len(new_record.Commitments) > COMMITMENT_HIST_SIZE {
		// Sort records by descending round order
		sort.Slice(new_record.Commitments, func(i, j int) bool {
			return new_record.Commitments[i].RoundNumber > new_record.Commitments[j].RoundNumber
		})
		new_record.Commitments = new_record.Commitments[:COMMITMENT_HIST_SIZE]
	}

	// Check for commitments collision and build SlashProof if necessary
	slashproof := aw.buildSlashProof(new_record)

	// Add record to table and the selected table index
	aw.table.records[PublicKeyHash(aw_peerID)] = new_record
	(*idx)[PublicKeyHash(aw_peerID)] = true
	aw.table.records_m.Unlock()

	return slashproof, nil
}

// Initialize the set of cryptographic keys needed for Aetherweave
func awKeys() (crypto.PrivKey, crypto.PubKey, *big.Int, *big.Int, error) {

	// Create sk, netpk, stakesk, stakeID
	log.Info("Generating new Aetherweave secret")

	// Generate 32 random bytes for secret
	var rand_buf [32]byte
	_, err := io.ReadFull(gorand.Reader, rand_buf[:])
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Convert random bytes to scalar sk and make sure it is in the Poseidon finite field
	// secret := new(big.Int).SetBytes(rand_buf[:])
	// secret = secret.Mod(secret, snarkconst.Q)

	// Generate key using secret
	// sk, netPK, err = crypto.BJJKeyPairFromScalar(secret)
	sk, netPK, err := crypto.GenerateBJJKeyPair(gorand.Reader)

	if err != nil {
		return sk, netPK, nil, nil, err
	}

	// Derive stakeSK and stakeID (32 bytes)
	sk_bjj, ok := sk.(*crypto.BJJPrivateKey)
	if !ok {
		return sk, netPK, nil, nil, errors.New("Failed to assert secret is a BJJ private key")
	}
	sk_bi := sk_bjj.SkToBigInt()
	sk_bim := sk_bi.Mod(sk_bi, snarkconst.Q)
	stakeSK, err := poseidon.Hash([]*big.Int{sk_bim})
	if err != nil {
		return sk, netPK, nil, nil, err
	}
	stakeID, err := poseidon.Hash([]*big.Int{stakeSK})
	if err != nil {
		return sk, netPK, nil, nil, err
	}

	return sk, netPK, stakeSK, stakeID, nil
}

// Generates ECDSA keys for an Ethereum wallet from the string of the private key
func ethKeys(privkey_str string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	var eth_privkey *ecdsa.PrivateKey
	var err error
	if privkey_str[:2] == "0x" {
		eth_privkey, err = gethcrypto.HexToECDSA(privkey_str[2:])
	} else {
		eth_privkey, err = gethcrypto.HexToECDSA(privkey_str)
	}
	// eth_privkey, err := gethcrypto.GenerateKey()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to generate ethereum wallet private key from provided string: %s", privkey_str))
	}

	eth_pubkey_i := eth_privkey.Public()
	eth_pubkey, ok := eth_pubkey_i.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("Failed to assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	return eth_privkey, eth_pubkey, nil
}

// Make stake deposit
func depositStake(client ethEng, contract awContract, eth_privkey *ecdsa.PrivateKey, eth_pubkey *ecdsa.PublicKey, stakeID *big.Int) error {

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return errors.Wrap(err, "Failed to get chain ID")
	}

	// Convert pubkey to Ethereum address
	walletAddress := gethcrypto.PubkeyToAddress(*eth_pubkey)

	// Get pending nonce for wallet
	nonce, err := client.PendingNonceAt(context.Background(), walletAddress)
	if err != nil {
		return errors.Wrap(err, "Failed to get pending nonce for wallet")
	}

	// Get gas price for transaction
	baseFee, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return errors.Wrap(err, "Failed to get suggested gas price")
	}

	// Get priority fee
	priorityFee, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	// Calculate maximum gas fee cap. Also add a priority fee
	increment := new(big.Int).Mul(big.NewInt(2), big.NewInt(gethparams.GWei))
	gasFeeCap := new(big.Int).Add(baseFee, increment)
	gasFeeCap = gasFeeCap.Add(gasFeeCap, priorityFee)

	// Prepare transaction options
	txOptsDeposit, err := bind.NewKeyedTransactorWithChainID(eth_privkey, chainID)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize deposit transaction")
	}
	txOptsDeposit.Nonce = new(big.Int).SetUint64(nonce)
	txOptsDeposit.Value = big.NewInt(STAKE_UNIT)
	txOptsDeposit.GasLimit = uint64(15000000)
	txOptsDeposit.GasPrice = gasFeeCap

	// Put up stake by depositing to the smart contract
	log.WithFields(logrus.Fields{"stakeID": stakeID, "nonce": txOptsDeposit.Nonce, "value": txOptsDeposit.Value, "gas limit": txOptsDeposit.GasLimit, "gas price": txOptsDeposit.GasPrice}).Info("Attempting deposit")
	txDeposit, err := contract.Deposit(txOptsDeposit, stakeID)
	if err != nil {
		return errors.Wrap(err, "Failed to make stake deposit to Aetherweave contract")
	}
	log.WithField("TX hash", txDeposit.Hash()).Info("Successfully made stake deposit to Aetherweave contract")

	// Query new wallet balance
	balance, err := client.BalanceAt(context.Background(), walletAddress, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to query wallet balance after making deposit")
	}
	log.WithFields(logrus.Fields{"wallet": walletAddress, "balance": balance}).Info("New wallet balance")

	return nil

}

// Initialize the smart contract. We need this function for testing. Won't be used in production
func initContract(client ethEng, contract awContract, eth_privkey *ecdsa.PrivateKey, eth_pubkey *ecdsa.PublicKey) error {

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return errors.Wrap(err, "Failed to get chain ID")
	}

	// Convert pubkey to Ethereum address
	walletAddress := gethcrypto.PubkeyToAddress(*eth_pubkey)

	// Get pending nonce for wallet
	nonce, err := client.PendingNonceAt(context.Background(), walletAddress)
	if err != nil {
		return errors.Wrap(err, "Failed to get pending nonce for wallet")
	}

	// Get gas price for transaction
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return errors.Wrap(err, "Failed to get suggested gas price")
	}

	// Prepare transaction options
	txOptsInit, err := bind.NewKeyedTransactorWithChainID(eth_privkey, chainID)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize contract init() transaction")
	}
	txOptsInit.Nonce = new(big.Int).SetUint64(nonce)
	txOptsInit.Value = big.NewInt(0)
	txOptsInit.GasLimit = uint64(5000000) // in units
	txOptsInit.GasPrice = gasPrice

	// Put up stake by depositing to the smart contract
	log.WithFields(logrus.Fields{"nonce": txOptsInit.Nonce, "value": txOptsInit.Value, "gas limit": txOptsInit.GasLimit, "gas price": txOptsInit.GasPrice}).Info("Attempting contract init")
	txInit, err := contract.Init(
		txOptsInit,
		TREE_DEPTH,
		new(big.Int).SetInt64(STAKE_UNIT),
		new(big.Int).SetInt64(AW_EPOCH_TIME),
		new(big.Int).SetInt64(ST_WITHDRAW_T),
		new(big.Int).SetInt64(ST_FREEZE_T),
	)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Aetherweave contract")
	}
	log.WithField("TX hash", txInit.Hash()).Info("Successfully initialized Aetherweave contract")

	return nil

}

// Initialize and return a new Aetherweave struct
func NewAetherweave(
	client ethEng,
	contract awContract,
	contractCaller awContractCaller,
	eth_privkey *ecdsa.PrivateKey,
	eth_pubkey *ecdsa.PublicKey,
	st_circ_path string,
	st_pkey_path string,
	st_vkey_path string,
	sh_circ_path string,
	sh_pkey_path string,
	sh_vkey_path string,
) (*Aetherweave, error) {

	// Initialize keys
	sk, netPK, stakeSK, stakeID, err := awKeys()
	if err != nil {
		return nil, err
	}

	// Marshal golibp2p publickey to bytes
	netPK_bytes, err := crypto.MarshalPublicKey(netPK)
	if err != nil {
		log.WithError(err).Error("Error marshalling pubkey")
		return nil, err
	}

	// Make initial stake deposit
	err = depositStake(client, contract, eth_privkey, eth_pubkey, stakeID)
	if err != nil {
		log.WithError(err).Error("Failed to deposit stake")
		return nil, err
	}

	// Load stake and share proof files
	files_bytes := make([][]byte, 6)
	for idx, fpath := range []string{st_circ_path, st_pkey_path, st_vkey_path, sh_circ_path, sh_pkey_path, sh_vkey_path} {
		file_bytes, err := os.ReadFile(fpath)
		if err != nil {
			log.WithError(err).WithField("fpath", fpath).Error("Failed to load ZK proof file")
			return nil, err
		}
		files_bytes[idx] = file_bytes
	}

	// Load witness calculators for stake and share proofs using WASM circuits
	st_circ_bytes, sh_circ_bytes := files_bytes[0], files_bytes[3]
	witness_calcs := make([]witness.Calculator, 2)
	for idx, wasmBytes := range [][]byte{st_circ_bytes, sh_circ_bytes} {
		wcalc, err := witness.NewCalculator(
			wasmBytes,
			witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
		)
		if err != nil {
			log.WithError(err).Error("Failed to load circuit wasm calculator")
			return nil, err
		}
		witness_calcs[idx] = wcalc
	}

	aw := &Aetherweave{
		table: &RecordsTable{
			idx_pub:      make(map[PublicKeyHash]bool),
			idx_priv:     make(map[PublicKeyHash]bool),
			nonce_pub:    Nonce(rand.NewGenerator().Uint64()),
			nonce_priv:   Nonce(rand.NewGenerator().Uint64()),
			records:      make(map[PublicKeyHash]*pb.PeerRecord),
			records_m:    sync.RWMutex{},
			blacklist:    make(map[PublicKeyHash]BlacklistEntry),
			blacklist_m:  sync.RWMutex{},
			served_com:   make(map[Hash]RoundNumber),
			served_com_m: sync.RWMutex{},
		},
		node_pubkey:  netPK_bytes,
		node_privkey: sk,
		eth_pubkey:   eth_pubkey,
		eth_privkey:  eth_privkey,
		stakeSK:      stakeSK,
		stakeID:      stakeID,
		sc_roots: &SCRootsTable{
			sc_table:   make(map[Hash]RoundNumber),
			sc_table_m: sync.RWMutex{},
		},
		round_number:    0,
		ethClient:       client,
		contract:        contract,
		contractCaller:  contractCaller,
		net_record_path: path.Join(cmd.DefaultDataDir(), "local_NetworkRecord.nr"),
		net_records_dir: path.Join(cmd.DefaultDataDir(), "imported_records"),
		st_circ_bytes:   files_bytes[0],
		st_wc:           witness_calcs[0],
		st_pkey_bytes:   files_bytes[1],
		st_vkey_bytes:   files_bytes[2],
		sh_circ_bytes:   files_bytes[3],
		sh_wc:           witness_calcs[1],
		sh_pkey_bytes:   files_bytes[4],
		sh_vkey_bytes:   files_bytes[5],

		st_cache_zkp:    &pb.ZKP{},
		st_cache_merkle: Hash{},
	}
	return aw, nil
}

func (s *Service) heartbeat(ctx context.Context) {

	queryChain(s.aw.ethClient, s.aw.contractCaller, s.aw.eth_pubkey)

	// Update the SC root map
	new_root := s.aw.updateSCRoots()
	log.WithFields(logrus.Fields{"roots_len": len(s.aw.sc_roots.sc_table), "new_root": new_root}).Info("Updated smart contract roots index.")

	// Generate nonce(s) for this round
	s.aw.table.nonce_pub = Nonce(rand.NewGenerator().Uint64())
	nonces := []Nonce{s.aw.table.nonce_pub}
	nonces_64 := []uint64{uint64(s.aw.table.nonce_pub)}
	if DUAL_TABLES {
		s.aw.table.nonce_priv = Nonce(rand.NewGenerator().Uint64())
		nonces = append(nonces, s.aw.table.nonce_priv)
		nonces_64 = append(nonces_64, uint64(s.aw.table.nonce_priv))
	}
	log.WithField("nonces", nonces).Info("Generated nonces.")

	// Build proof of stake
	var proof_of_stake *pb.ZKP
	var merkle_root Hash
	var err error

	if ST_CACHE && !new_root && len(s.aw.st_cache_zkp.GetPiB().GetPoints()) == 3 {
		log.Info("No SC root update. Using cached proof of stake.")
		proof_of_stake = s.aw.st_cache_zkp
		merkle_root = s.aw.st_cache_merkle
	} else {
		log.Info("Building proof of stake")
		proof_of_stake, merkle_root, err = s.aw.build_proof_of_stake()

		if err != nil {

			maxDelay := calculateRoundStartTime(int64(s.aw.round_number)) + AW_CUTOFF_SPAN - time.Now().Unix()
			if maxDelay <= 0 {
				log.WithField("maxDelay", maxDelay).Warn("Time slip ocurred. Skipping this heartbeat")
				return
			}
			deposit_time := time.Now().Unix() + rand.NewGenerator().Int63n(maxDelay)
			delta := time.Until(time.Unix(deposit_time, 0))
			log.WithError(err).WithField("sleepTime", delta).Error("Failed to build proof of stake for heartbeat. Attempting to make deposit again.")
			time.Sleep(delta)
			err = depositStake(s.aw.ethClient, s.aw.contract, s.aw.eth_privkey, s.aw.eth_pubkey, s.aw.stakeID)
			if err != nil {
				log.WithError(err).Error("Failed to deposit stake")
				return
			}

			// Try building proof of stake again
			proof_of_stake, merkle_root, err = s.aw.build_proof_of_stake()
			if err != nil {
				log.WithError(err).Error("Failed to build proof of stake again. Retrying next round.")
				return
			}

		}
		s.aw.st_cache_zkp = proof_of_stake
		s.aw.st_cache_merkle = merkle_root
	}

	// Get latest local node multiaddrs
	local_multiaddrs, err := s.cfg.p2p.GetMultiAddrs()
	if err != nil {
		log.WithError(err).Error("Failed to get multiaddrs")
		return
	}
	log.WithField("local_multiaddrs", local_multiaddrs).Info("Fetched local multiaddrs")

	// Build node network record
	node_net_record, err := build_network_record(s.aw.node_privkey, proof_of_stake, local_multiaddrs, merkle_root)
	if err != nil {
		log.WithError(err).Error("Failed to build our network record")
		return
	}
	log.Info("Built local NetworkRecord")

	// Write out NetworkRecord
	if err := writeLocalNetworkRecord(node_net_record, s.aw.net_record_path); err != nil {
		log.WithError(err).Error("Failed to write out local NetworkRecord")
	} else {
		log.WithField("path", s.aw.net_record_path).Info("Wrote network record file")
	}

	// Add our own NetworkRecord to our public table
	aw_peerID, err := peer.IDFromPublicKey(s.aw.node_privkey.GetPublic())
	if err != nil {
		log.WithError(err).Error("Failed to get local aw_peerID")
		return
	}
	s.aw.table.records_m.Lock()
	s.aw.table.records[PublicKeyHash(aw_peerID)] = &pb.PeerRecord{
		NetRecord:   node_net_record,
		Commitments: make([]*pb.CommitmentRecord, 0),
	}
	s.aw.table.idx_pub[PublicKeyHash(aw_peerID)] = true
	s.aw.table.records_m.Unlock()

	// Print peerstore status
	// s.logPeerStoreStatus()

	// Sample random peerIDs of peers in our table
	// If we are using DUAL_TABLES, we sample the private table index
	public_keys, aw_peerIDs, err := s.aw.table.samplePublicKeys(uint(AW_REQ_NUM), !DUAL_TABLES, map[PublicKeyHash]bool{PublicKeyHash(aw_peerID): true})
	if err != nil {
		log.WithError(err).Error("Failed to sample records table")
		return
	}
	log.WithField("aw_peerIDs count", len(aw_peerIDs)).Info("Sampled Aetherweave table")

	// Build CommitmentOpenings for sampled peers
	log.Info("Building commitments")
	comm_record, comm_openings, err := build_commitments(public_keys, s.aw.round_number, s.aw.node_privkey, s.aw.stakeID, s.aw.sh_wc, s.aw.sh_pkey_bytes)
	if err != nil {
		log.WithError(err).Error("Failed to build commitments")
		return
	}

	// Build requests to peers
	log.Info("Building requests")
	requests := make([]*pb.Request, len(comm_openings))
	for i, comm_opening := range comm_openings {
		requests[i] = &pb.Request{
			Nonces:            nonces_64,
			SenderRecord:      node_net_record,
			CommitmentRecord:  comm_record,
			CommitmentOpening: comm_opening,
			Signature:         &pb.Signature{},
		}

		// Sign request
		err := signAWMessage(requests[i], s.aw.node_privkey)
		if err != nil {
			log.WithError(err).Error("Failed to sign request")
		}
	}

	// Pick times at which to make the requests
	round_cutoff_limit := calculateRoundStartTime(int64(s.aw.round_number)) + AW_CUTOFF_SPAN
	current_time := time.Now().Unix()
	remaining_time := round_cutoff_limit - current_time
	if remaining_time <= 0 {
		log.Warn("No time remaining for staggered requests. Ending heartbeat early.")
		return
	}
	req_sleep_times := make([]int64, len(requests))
	for idx := range req_sleep_times {
		req_sleep_times[idx] = current_time + rand.NewGenerator().Int63n(remaining_time)
	}
	sort.Slice(req_sleep_times, func(i, j int) bool { return req_sleep_times[i] < req_sleep_times[j] })

	// Make staggered requests
	log.WithFields(logrus.Fields{"current_ts": current_time, "req_times": req_sleep_times}).Info("Starting staggered requests")
	wg := new(sync.WaitGroup)
	for idx, request := range requests {
		sleep := time.Until(time.Unix(req_sleep_times[idx], 0))
		if sleep > 0 {
			log.WithFields(logrus.Fields{"sleep": sleep, "idx": idx}).Info("Sleeping before request")
			time.Sleep(sleep)
		}

		// Unmarshal peer multiaddr
		log.WithFields(logrus.Fields{"idx": idx}).Info("Waking to continue request")
		s.aw.table.records_m.RLock()
		peer_multiaddr, err := ma.NewMultiaddrBytes(s.aw.table.records[aw_peerIDs[idx]].GetNetRecord().GetMultiaddr().GetMultiaddr())
		s.aw.table.records_m.RUnlock()
		if err != nil {
			log.WithFields(logrus.Fields{"function": "heartbeat", "aw_peerID": aw_peerIDs[idx]}).WithError(err).Error("Could not unmarshall peer addrinfo")
			continue
		}

		// Get the native peerID from the multiaddr
		peerID, err := GetPeerIDFromMultiaddr(peer_multiaddr)
		if err != nil {
			log.WithField("peerID", peerID).Error("Failed to GetPeerIDFromMultiaddr() for request")
			continue
		}

		// Ensure an address entry for this peer exists in the peer store
		if _, err := s.cfg.p2p.Peers().Address(peerID); err != nil {
			log.WithFields(logrus.Fields{"aw_peerID": aw_peerID, "peerID": peerID}).Info("Adding aetherweave peer to peer handler")
			s.cfg.p2p.Host().Peerstore().AddAddr(peerID, peer_multiaddr, time.Minute*15)
			s.cfg.p2p.Peers().Add(nil, peerID, peer_multiaddr, network.DirUnknown)
		}

		wg.Add(1)

		go func(ctx context.Context, request *pb.Request, peerID peer.ID, nonces []Nonce) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				log.Warn("Heartbeat context has ended. Stopping request.")
				return
			default:
			}
			log.WithFields(logrus.Fields{"peerID": peerID}).Info("Making request")
			err := s.sendRPCAWRequest(ctx, request, peerID, nonces)
			if err != nil {
				log.WithError(err).Error("Failed to make Aetherweave Request RPC")
			}
		}(ctx, request, peerID, nonces)
	}

	// Wait for all requests to complete
	wg.Wait()

	// Maintain RecordsTable and remove low scoring peers
	del_idxs, del_recs := s.aw.table.maintainRecordsTable(s.aw.node_pubkey, true)
	log.WithFields(logrus.Fields{"table": "public", "deleted_idxs": del_idxs, "deleted_records": del_recs}).Info("Maintained records table.")
	if DUAL_TABLES && len(nonces) == 2 {
		s.aw.table.maintainRecordsTable(s.aw.node_pubkey, false)
		log.WithFields(logrus.Fields{"table": "private", "deleted_idxs": del_idxs, "deleted_records": del_recs}).Info("Maintained records table.")
	}

	// Get old slashees whose slashing time has expired
	log.Info("Maintaining peer blacklist")
	old_slashees := []PublicKeyHash{}
	s.aw.table.blacklist_m.RLock()
	for peer_pk_hash, slash := range s.aw.table.blacklist {
		if slash.timestamp.Add(BL_TIME * time.Second).Before(time.Now()) {
			old_slashees = append(old_slashees, peer_pk_hash)
		}
	}
	s.aw.table.blacklist_m.RUnlock()

	// Remove old slashes
	if len(old_slashees) > 0 {
		log.WithField("old_slashees", old_slashees).Info("Removing old slashees from blacklist")
		s.aw.table.blacklist_m.Lock()
		for _, slashee := range old_slashees {
			delete(s.aw.table.blacklist, slashee)
		}
		s.aw.table.blacklist_m.Unlock()
	}

	// Remove old served Commitments entries
	removed_comms := s.aw.table.maintainServedCommitmentsTable(s.aw.round_number)
	log.WithField("removed_commitments", removed_comms).Info("Maintained served commitments table")

}

// Function to bootstrap Aetherweave tables after cold boot. If max_records is > 0, limit the number of imported records.
func (s *Service) bootstrapFromLocalNetRec(max_records int) {
	log.Info("Bootstrapping tables from local network records")
	// Load local NetworkRecord files if they exist and add them to the table
	imported_net_recs, err := readLocalNetworkRecords(s.aw.net_records_dir)
	if err != nil {
		log.WithError(err).Error("Failed to import network records")
	}
	for record_idx, net_record := range imported_net_recs {
		// If max_records is > 0, limit the number of imported records.
		if max_records > 0 && record_idx > max_records {
			break
		}

		_, aw_peerID, err := processMarshalledPubkey(net_record.GetPublicKey().GetPubkey())
		if err != nil {
			log.WithError(err).Error("Failed to get peerID from public key")
			continue
		}
		// Skip record if it's already in the table
		s.aw.table.records_m.RLock()
		_, ok := s.aw.table.records[PublicKeyHash(aw_peerID)]
		s.aw.table.records_m.RUnlock()
		if ok {
			log.WithField("aw_peerID", aw_peerID).Info("Skipping adding peer from file. Record already exists")
			continue
		} else {

			// Unmarshal peer multiaddr
			peer_multiaddr, err := ma.NewMultiaddrBytes(net_record.GetMultiaddr().GetMultiaddr())
			if err != nil {
				log.WithFields(logrus.Fields{"function": "bootstrapFromLocalNetRec", "aw_peerID": aw_peerID}).WithError(err).Error("Could not unmarshall peer addrinfo")
				continue
			}

			// Get the native peerID from the multiaddr
			peerID, err := GetPeerIDFromMultiaddr(peer_multiaddr)
			if err != nil {
				log.WithField("multiaddr", peer_multiaddr).Error("Failed to GetPeerIDFromMultiaddr()")
				continue
			}

			// Add record to table
			s.aw.table.records_m.Lock()
			s.aw.table.records[PublicKeyHash(aw_peerID)] = &pb.PeerRecord{
				NetRecord:   net_record,
				Commitments: make([]*pb.CommitmentRecord, 0),
			}
			// Update table indexes depending on if we're using DUAL_TABLES
			s.aw.table.idx_pub[PublicKeyHash(aw_peerID)] = true
			if DUAL_TABLES {
				s.aw.table.idx_priv[PublicKeyHash(aw_peerID)] = true
			}
			s.aw.table.records_m.Unlock()

			log.WithFields(logrus.Fields{"aw_peerID": aw_peerID, "peerID": peerID, "multiaddr": peer_multiaddr.String()}).Info("Added peer from file")

			// Ignore nodes that are already connected.
			if s.cfg.p2p.Host().Network().Connectedness(peerID) == network.Connected {
				log.WithFields(logrus.Fields{"function": "bootstrapFromLocalNetRec", "aw_peerID": aw_peerID, "peerID": peerID}).Info("Skipping connection to already connected peer")
				continue
			}

			// Add peer to peer handler
			log.WithFields(logrus.Fields{"aw_peerID": aw_peerID, "peerID": peerID}).Info("Adding aetherweave peer to peer handler")
			s.cfg.p2p.Host().Peerstore().AddAddr(peerID, peer_multiaddr, time.Minute*15)
			s.cfg.p2p.Peers().Add(nil, peerID, peer_multiaddr, network.DirUnknown)

			// Try to connect to peer
			err = s.connectWithPeer(s.cfg.p2p.GetContext(),
				peer.AddrInfo{
					ID:    peerID,
					Addrs: []ma.Multiaddr{peer_multiaddr},
				})
			if err != nil {
				log.WithFields(logrus.Fields{"aw_peerID": aw_peerID, "peerID": peerID}).WithError(err).Error("Failed to connectWithPeer()")
			} else {
				log.WithFields(logrus.Fields{"aw_peerID": aw_peerID, "peerID": peerID}).Info("Connected to peer from disk NetworkRecord")
				s.cfg.p2p.Peers().Add(nil, peerID, peer_multiaddr, network.DirOutbound)
			}
		}
	}
}

// aetherweaveScheduler schedules periodically running heartbeats
func (s *Service) aetherweaveScheduler() {

	log.WithField("AW_NODES_SQ", AW_NODES_SQ).Info("Setting number of nodes")
	executed_heartbeats := 0
	warmup_hbs := 20
	finished_warmup := false

	// Run a heartbeat every round
	for {

		// Bootstrap tables using locally stored network records if necessary
		s.aw.table.records_m.RLock()
		table_size := len(s.aw.table.records)
		s.aw.table.records_m.RUnlock()
		if table_size < 2 {
			s.bootstrapFromLocalNetRec(TABLE_SIZE)
		}

		// Update round number
		s.aw.round_number = RoundNumber(calculateRound())

		// Check next round start time
		next_round_start := calculateRoundStartTime(int64(s.aw.round_number + 1))

		// Check timespan up until next round
		remaining_time := next_round_start - time.Now().Unix()

		// Sleep and continue in next round if there's not enough time left in the round
		if remaining_time < (AW_ROUND_TIME - AW_CUTOFF_SPAN) {
			log.Info("Not enough time in current round, sleeping until next")
			sleep_duration := next_round_start - time.Now().Unix()
			time.Sleep(time.Duration(sleep_duration) * time.Second)
			continue
		}

		// Log current status
		activePeers := s.cfg.p2p.Peers().Active()
		activePeerCount := uint(len(activePeers))
		s.aw.table.records_m.RLock()
		idx_len := len(s.aw.table.idx_pub)
		s.aw.table.records_m.RUnlock()
		log.WithFields(logrus.Fields{"aw_round": s.aw.round_number, "aw_public_idx_size": idx_len, "activePeerCount": activePeerCount}).Info("Starting heartbeat")

		// Run heartbeat
		ctx, cancel := context.WithTimeout(s.ctx, time.Duration(AW_ROUND_TIME)*time.Second)
		hb_start := time.Now().Unix()
		s.heartbeat(ctx)
		cancel()
		hb_duration := time.Now().Unix() - hb_start
		log.WithField("duration_s", hb_duration).Info("Heartbeat Complete")
		warmup_hbs -= 1
		if warmup_hbs < 0 {

			executed_heartbeats += 1
			// Adjust table sizes if needed
			if executed_heartbeats%32 == 0 {

				// If this is the first time running this after the warmup, no need to stop the profiler and update AW_NODES_SQ
				if !finished_warmup {
					finished_warmup = true
				} else {
					// Stop active CPU profiler, if any
					if CPU_PROFILING {
						pprof.StopCPUProfile()
					}

					// Ceil(Square root(Number of nodes in the protocol))
					AW_NODES_SQ += 5
				}

				log.WithField("AW_NODES_SQ", AW_NODES_SQ).Info("Setting number of nodes")

				// Number of nodes participating in the protocol
				AW_NODES_NUM = AW_NODES_SQ * AW_NODES_SQ

				// Maximum number of peers to keep in our records table
				TABLE_SIZE = AW_SCALE * AW_NODES_SQ

				// Number of peers to which we make requests every round
				AW_REQ_NUM = TABLE_SIZE

				// Bootstrap tables to have as many records as new size
				s.bootstrapFromLocalNetRec(TABLE_SIZE)

				// Create new CPU profile file and start profiling
				if CPU_PROFILING {
					f, err := os.Create(path.Join(cmd.DefaultDataDir(), fmt.Sprintf("cpu_%d.prof", AW_NODES_SQ)))
					if err != nil {
						log.Fatal(err)
					}
					err = pprof.StartCPUProfile(f)
					if err != nil {
						log.Fatal(err)
					}
				}

			}
		}

		if AW_NODES_SQ > 25 {
			log.Info("Finished Experiments")
			// Stop profiling
			if CPU_PROFILING {
				pprof.StopCPUProfile()
			}
			return
		}

		// Sleep until next round
		sleep_duration := next_round_start - time.Now().Unix()
		log.WithField("sleep_time", sleep_duration).Info("Heartbeat done. Sleeping until next one.")
		time.Sleep(time.Duration(sleep_duration) * time.Second)
	}

}

func (s *Service) connectWithPeer(ctx context.Context, info peer.AddrInfo) error {
	ctx, span := trace.StartSpan(ctx, "p2p.connectWithPeer")
	defer span.End()

	if info.ID == s.cfg.p2p.Host().ID() {
		return nil
	}
	if err := s.cfg.p2p.Peers().IsBad(info.ID); err != nil {
		return errors.Wrap(err, "refused to connect to bad peer")
	}
	ctx, cancel := context.WithTimeout(ctx, params.BeaconConfig().RespTimeoutDuration())
	defer cancel()
	if err := s.cfg.p2p.Host().Connect(ctx, info); err != nil {
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(info.ID)
		return err
	}
	return nil
}

// signMessageLibp2p generates and signs a SHA256 digest of the given message with
// the given private key and returns a 64-byte (r || s) signature.
func signMessageLibp2p(privkey crypto.PrivKey, msg []byte) ([]byte, error) {
	signature, err := privkey.Sign(msg)
	if len(signature) > 71 {
		return nil, errors.New(fmt.Sprintf("Signature must be <=71 bytes, instead it's: %d", len(signature)))
	}
	return signature, err
}

// Sign an Aetherweave message
func signAWMessage(m Signable, priv crypto.PrivKey) error {

	// Use zeros in the signature field
	s_field := m.GetSignature()
	s_field.Signature = make([]byte, SIG_SIZE)

	// Serialize data to sign: we sign the canonical (ssz/protobuf) encoding of the record minus the signature
	response_bytes, err := m.MarshalSSZ()
	if err != nil {
		log.WithError(err).Errorf("Failed to marshal message for signing %v", m)
		return errors.Wrapf(err, "Failed to marshal message for signing")
	}

	// Sign the message
	signature, err := signMessageLibp2p(priv, response_bytes)
	if err != nil {
		log.WithError(err).Errorf("Failed to sign message %v", m)
		return err
	}
	s_field.Signature = signature

	return nil
}

// Verify that an Aetherweave message was signed using the given public key
func verifyAWMessage(m Signable, pub crypto.PubKey) (bool, error) {
	// Temporarily replace signature with zeros for verification
	init_signature := m.GetSignature().Signature
	m.GetSignature().Signature = make([]byte, SIG_SIZE)

	// Marshal message for signature verification
	m_bytes, err := m.MarshalSSZ()
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal message for signing")
	}

	// Verify sig with sender's public key
	ok, err := pub.Verify(m_bytes, init_signature)
	if err != nil || !ok {
		return false, errors.Wrap(err, "invalid commitment record signature")
	}

	// Restore signature
	m.GetSignature().Signature = init_signature

	return true, nil
}

// Builds and returns the serialized share proof. Does NOT include public signals
func build_share_proof(node_privkey crypto.PrivKey, stakeID *big.Int, rootHash Hash, share *big.Int, round_number RoundNumber, sh_wc witness.Calculator, sh_pkey_bytes []byte) (*pb.ZKP, error) {

	// Prepare and serialize inputs to generate ZKP
	inputBytes, err := prepareShareProofInput(node_privkey, stakeID, rootHash, share, round_number)
	if err != nil {
		log.WithError(err).Error("Failed to prepare share proof input data")
		return nil, err
	}

	// Build ZKP
	// log.WithFields(logrus.Fields{"function": "generateZKProof", "RSS": getRSS()}).Info("RSS Before")
	zkp, err := generateZKProof(inputBytes, sh_wc, sh_pkey_bytes)
	if err != nil {
		log.WithError(err).Error("Failed to create share proof ZKP")
		return nil, err
	}
	// log.WithFields(logrus.Fields{"function": "generateZKProof", "RSS": getRSS()}).Info("RSS After")

	// Build ZKP protobuf message
	// The receiver already has or can derive the public signals from the other protobuf messages
	zkp_pb, err := ProofDataToZKP(zkp.Proof)
	if err != nil {
		log.WithError(err).Error("Failed to convert ZKP ProofData to ZKP protobuf")
		return nil, err
	}

	return zkp_pb, nil
}

func build_commitments(peers []PublicKey, round_number RoundNumber, node_privkey crypto.PrivKey, stakeID *big.Int, sh_wc witness.Calculator, sh_pkey_bytes []byte) (*pb.CommitmentRecord, []*pb.CommitmentOpening, error) {
	if len(peers) == 0 {
		return nil, nil, errors.New("no peers provided for commitments")
	}

	// Sort public keys lexicographically to ensure deterministic ordering
	sort.Slice(peers, func(i, j int) bool {
		return string(peers[i]) < string(peers[j])
	})

	// Convert peers to [][]byte
	leafData := make([][]byte, len(peers))
	for i, pk := range peers {
		leafData[i] = pk
	}

	// Build Merkle tree
	tree, err := build_merkle_tree(leafData)
	if err != nil {
		log.WithError(err).Error("Failed to build merkle tree for commitment")
		return nil, nil, err
	}
	rootHash := tree[len(tree)-1][0] // Root of the Merkle tree

	// Calculate share
	share, err := calculate_share(node_privkey, round_number, rootHash)
	if err != nil {
		log.WithError(err).Error("Failed to calculate share for commitment record")
		return nil, nil, err
	}

	// Convert share to bytes and add zero-padding at the start
	share_bytes := make([]byte, 32)
	copyStart := 32 - len(share.Bytes())
	if copyStart < 0 {
		// This should not happen - share is < 256 bits
		return nil, nil, errors.New("Modulo result exceeds 32 bytes")
	}
	copy(share_bytes, share.Bytes()[copyStart:])

	// Calculate share proof
	shareProof, err := build_share_proof(node_privkey, stakeID, rootHash, share, round_number, sh_wc, sh_pkey_bytes)
	if err != nil {
		log.WithError(err).Error("Failed to build share proof")
		return nil, nil, err
	}

	// Create CommitmentRecord
	commitment := &pb.CommitmentRecord{
		RootHash:    &pb.Hash{Hash: rootHash[:]},
		RoundNumber: uint64(round_number),
		SlashShare:  share_bytes,
		ShareProof:  shareProof,
	}

	// Build CommitmentOpenings for each peer
	openings := make([]*pb.CommitmentOpening, len(peers))
	for i := range peers {
		proofHashes, err := build_merkle_proof(i, tree)
		if err != nil {
			return nil, nil, err
		}

		pbProof := make([]*pb.Hash, len(proofHashes))
		for j, h := range proofHashes {
			pbProof[j] = &pb.Hash{Hash: h[:]}
		}

		openings[i] = &pb.CommitmentOpening{
			ParentHash: commitment.RootHash,
			LeafIndex:  uint32(i),
			Proof:      pbProof,
		}
	}

	return commitment, openings, nil
}

// Fetch the stake merkle proof from the blockchain for the given stakeID
func getStakeMerkleProof(contract awContractCaller, stakeID *big.Int) (*awcontract.SparseMerkleTreeProof, error) {

	// Call the contract to get merkle proof
	opts := &bind.CallOpts{
		Context: context.Background(),
	}
	proof, err := contract.GetProof(opts, stakeID)
	if err != nil {
		log.WithError(err).Error("Failed to call AetherWeavePrivate:GetProof()")
		return nil, err
	}

	return &proof, nil
}

// Builds StakeProof - the zero knowledge proof that demonstrates ownership of stake (does NOT include public signals).
// Also returns Merkle Root necessray for verifying the ZKP
func (aw *Aetherweave) build_proof_of_stake() (*pb.ZKP, Hash, error) {

	// Fetch stake merkle proof for the node's stakeID from the smart contract
	merkle_proof, err := getStakeMerkleProof(aw.contractCaller, aw.stakeID)
	if err != nil {
		log.WithError(err).Error("Failed to get stake merkle proof")
		return nil, Hash{}, err
	}

	// Prepare serialized JSON inputs to construct the Stake Proof
	inputsBytes, err := prepareStakeProofInput(aw.node_privkey, aw.stakeID, merkle_proof)
	if err != nil {
		log.WithError(err).Error("Failed prepare marshalled JSON input for stake proof ZKP")
		return nil, merkle_proof.Root, err
	}

	// Build ZKP
	// Public signals are decimal strings of [netPK.X, netPK.Y, merkle root]
	zkp, err := generateZKProof(inputsBytes, aw.st_wc, aw.st_pkey_bytes)
	if err != nil {
		log.WithError(err).Error("Failed to create proof of stake ZKP")
		return nil, merkle_proof.Root, err
	}

	// Build ZKP protobuf message
	zkp_pb, err := ProofDataToZKP(zkp.Proof)
	if err != nil {
		log.WithError(err).Error("Failed to convert ZKP ProofData to ZKP protobuf")
		return nil, merkle_proof.Root, err
	}

	// Serialize ZKP JSON
	// zkp_json_b, err := json.Marshal(*zkp.Proof)
	// if err != nil {
	// 	log.WithError(err).Error("Failed to marshal stake ZKP")
	// 	return nil, merkle_proof.Root, err
	// }

	return zkp_pb, merkle_proof.Root, nil
}

// Calculate share for a commitment root
func calculate_share(secret crypto.PrivKey, round_number RoundNumber, commitmentRoot Hash) (*big.Int, error) {

	// Get BJJ private key
	bjj_secret, ok := secret.(*crypto.BJJPrivateKey)
	if !ok {
		return nil, errors.New("Failed to assert that key is BJJ")
	}
	bjj_secret_bi := bjj_secret.SkToBigInt()

	// Commitment root in big.Int format
	commitment_root_bi := new(big.Int).SetBytes(commitmentRoot[:])

	// Round number in big.Int format
	round_number_bi := new(big.Int).SetUint64(uint64(round_number))

	// Share poseidon hashing constant
	poseidon_c_bi := new(big.Int).SetInt64(SP_CONST)
	alpha, err := poseidon.Hash([]*big.Int{bjj_secret_bi, round_number_bi, poseidon_c_bi})
	if err != nil {
		return nil, err
	}

	stake_secret, err := poseidon.Hash([]*big.Int{bjj_secret_bi})
	if err != nil {
		return nil, err
	}

	share := new(big.Int).Mul(alpha, commitment_root_bi)
	share = share.Add(share, stake_secret)
	share = share.Mod(share, snarkconst.Q)

	return share, nil
}

// build_network_record creates and signs a NetworkRecord for the local node.
func build_network_record(privkey crypto.PrivKey, pos *pb.ZKP, multiaddrs []ma.Multiaddr, merkle_root Hash) (*pb.NetworkRecord, error) {

	// Get current timestamp
	timestamp := uint64(time.Now().Unix())

	// Pick a multiaddr and marshal it
	if len(multiaddrs) == 0 {
		return nil, errors.New("No available multiaddrs")
	}

	// Use 0 as the index to get the TCP address, or len(multiaddrs)-1 to get the QUIC one
	localMultiaddr := multiaddrs[0]

	localMultiaddr_b, err := localMultiaddr.MarshalBinary()
	if err != nil || len(localMultiaddr_b) == 0 {
		log.WithError(err).WithField("len", len(localMultiaddr_b)).Error("Error marshalling local multiaddr")
		return nil, err
	}

	// Marshal golibp2p publickey to bytes
	c_pubKeyBytes, err := crypto.MarshalPublicKey(privkey.GetPublic())
	if err != nil {
		log.WithError(err).Error("Error marshalling pubkey")
		return nil, err
	}

	// Build NetworkRecord with a blank signature first
	nr := &pb.NetworkRecord{
		PublicKey:    &pb.PublicKey{Pubkey: c_pubKeyBytes},
		ProofOfStake: pos,
		MerkleRoot:   &pb.Hash{Hash: merkle_root[:]},
		Multiaddr:    &pb.AWMultiAddr{Multiaddr: localMultiaddr_b},
		Timestamp:    timestamp,
		Signature:    &pb.Signature{},
	}

	// Sign NetworkRecord
	err = signAWMessage(nr, privkey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign NetworkRecord")
	}

	return nr, nil
}

// verify_commitment_opening checks whether a public key matches the root hash via the given Merkle proof.
func verify_commitment_opening(opening *pb.CommitmentOpening, nodePK PublicKey) (bool, error) {
	if opening == nil || opening.ParentHash == nil {
		return false, errors.New("invalid CommitmentOpening: missing fields")
	}

	// Start with hash of public key
	runningHash := sha256modQ(nodePK)
	currentIdx := opening.LeafIndex

	// Iterate through proof hashes
	for _, sibling := range opening.Proof {
		if sibling == nil {
			return false, errors.New("invalid sibling hash in proof")
		}
		var combined []byte
		if currentIdx%2 == 0 {
			combined = append(runningHash[:], sibling.Hash...)
		} else {
			combined = append(sibling.Hash, runningHash[:]...)
		}
		runningHash = sha256modQ(combined)
		currentIdx /= 2
	}

	// Compare final hash to root
	expectedRoot := opening.ParentHash.Hash
	return bytes.Equal(runningHash[:], expectedRoot), nil
}

// Utility function to calculate SHA256 sum and use modulo Q to put it in the range supported by SNARKs
func sha256modQ(data []byte) [32]byte {
	hash := sha256.Sum256(data)
	hash_bi := new(big.Int).SetBytes(hash[:])
	hash_bi = hash_bi.Mod(hash_bi, snarkconst.Q)
	hash_bytes := hash_bi.Bytes()
	var hash_buff [32]byte
	// Calculate the starting position for the copy to ensure zero-padding at the start
	copyStart := 32 - len(hash_bytes)
	if copyStart < 0 {
		// This should not happen - Q is < 256 bits
		log.Error("Modulo result exceeds 32 bytes")
	}
	copy(hash_buff[copyStart:], hash_bytes)
	return hash_buff
}

// build_merkle_tree builds a Merkle tree from input byte slices.
// The tree is returned as a list of levels (bottom-up), each level containing Hashes.
func build_merkle_tree(data [][]byte) ([][]Hash, error) {
	if len(data) == 0 {
		return nil, errors.New("no data to build merkle tree")
	}

	var tree [][]Hash

	// Create leaf level
	leafLevel := make([]Hash, len(data))
	for i, item := range data {
		hash := sha256modQ(item)
		leafLevel[i] = hash
	}
	tree = append(tree, leafLevel)

	// Build tree levels
	level := leafLevel
	for len(level) > 1 {
		var nextLevel []Hash
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			// If right sibling does not exist, duplicate left
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			combined := append(left[:], right[:]...)
			parent := sha256modQ(combined)
			nextLevel = append(nextLevel, parent)
		}
		tree = append(tree, nextLevel)
		level = nextLevel
	}

	return tree, nil
}

// Build a Merkle proof for a given leaf index, and tree built using build_merkle_tree.
func build_merkle_proof(leafIdx int, tree [][]Hash) ([]Hash, error) {
	if len(tree) == 0 || leafIdx < 0 || leafIdx >= len(tree[0]) {
		return nil, errors.New("invalid tree or leaf index")
	}

	var proof []Hash
	currentIdx := leafIdx

	for level := 0; level < len(tree)-1; level++ {
		nodes := tree[level]
		if len(nodes) == 1 {
			break // reached root
		}

		var sibling Hash
		if currentIdx%2 == 0 {
			// If current is even, sibling is current + 1 if it exists, else current
			if currentIdx+1 < len(nodes) {
				sibling = nodes[currentIdx+1]
			} else {
				// Duplicate if no sibling
				sibling = nodes[currentIdx]
			}
		} else {
			// If current is odd, sibling is current - 1
			sibling = nodes[currentIdx-1]
		}

		proof = append(proof, sibling)
		currentIdx /= 2
	}

	return proof, nil
}

// score computes a float64 score between 0.0 and 1.0 by hashing src_pubkey + rec_pubkey + nonce.
func score(src_pubkey PublicKey, rec_pubkey PublicKey, nonce Nonce) Score {
	// Concatenate pubkeys and nonce
	combined := append([]byte{}, src_pubkey...)
	combined = append(combined, rec_pubkey...)

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	combined = append(combined, nonceBytes...)

	// Compute SHA256 digest
	hasher := murmur3.New64()
	hasher.Write(combined)
	hashValue := hasher.Sum64()

	// Normalize to [0.0, 1.0]
	return Score(float64(float64(hashValue) / float64(math.MaxUint64)))
}

// Return the first SlashProof we can build for a PeerRecord
func (aw *Aetherweave) buildSlashProof(peerRecord *pb.PeerRecord) *pb.SlashProof {
	if peerRecord == nil || peerRecord.NetRecord == nil || peerRecord.NetRecord.PublicKey == nil {
		return nil
	}

	slashee := peerRecord.NetRecord.PublicKey
	records := peerRecord.Commitments

	// Map: round_number -> previously seen commitment for that round
	roundMap := make(map[uint64]*pb.CommitmentRecord)

	for _, r1 := range records {
		if r1 == nil || r1.RootHash == nil {
			continue
		}
		round := r1.RoundNumber
		if r2, exists := roundMap[round]; exists {
			if !bytes.Equal(r1.RootHash.Hash, r2.RootHash.Hash) {
				proof := &pb.SlashProof{
					Slashee:  slashee,
					Record_1: r1,
					Record_2: r2,
				}
				return proof
			}
		} else {
			roundMap[round] = r1
		}
	}

	return nil
}

// Calculate StakeSk from two shares in a SlashProof
func calculateStakeSk(share1 *big.Int, commitment1 *big.Int, share2 *big.Int, commitment2 *big.Int) *big.Int {
	x1y2 := new(big.Int).Mul(commitment1, share2)
	x2y1 := new(big.Int).Mul(commitment2, share1)
	x1x2_diff := new(big.Int).Sub(commitment1, commitment2)
	prod_diff := new(big.Int).Sub(x1y2, x2y1)
	stakeSk := new(big.Int).Div(prod_diff, x1x2_diff)
	return stakeSk
}

// Processes SlashProofs and submits them to the blockchain for slashing
func submitSlashProofs(client ethEng, contract awContract, proofs []*pb.SlashProof, eth_pubkey *ecdsa.PublicKey, eth_privkey *ecdsa.PrivateKey) error {

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return errors.Wrap(err, "Failed to get chain ID")
	}

	// Convert pubkey to Ethereum address
	walletAddress := gethcrypto.PubkeyToAddress(*eth_pubkey)

	// Iterate through slashproofs
	for _, proof := range proofs {

		// Calculate stakeSk to be slashed
		share1 := new(big.Int).SetBytes(proof.Record_1.SlashShare)
		commitment1 := new(big.Int).SetBytes(proof.Record_1.RootHash.Hash)
		share2 := new(big.Int).SetBytes(proof.Record_2.SlashShare)
		commitment2 := new(big.Int).SetBytes(proof.Record_2.RootHash.Hash)
		stakeSk := calculateStakeSk(share1, commitment1, share2, commitment2)

		// Derive stakeID
		stakeSk_b := sha256.Sum256(stakeSk.Bytes())
		stakeID := new(big.Int).SetBytes(stakeSk_b[:])

		// Get pending nonce for wallet
		nonce, err := client.PendingNonceAt(context.Background(), walletAddress)
		if err != nil {
			log.WithError(err).Error("Failed to get pending nonce for wallet")
		}

		// Get gas price for transaction
		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.WithError(err).Error("Failed to get suggested gas price")
		}

		// Prepare transaction options
		txOptsSlash, err := bind.NewKeyedTransactorWithChainID(eth_privkey, chainID)
		if err != nil {
			return errors.Wrap(err, "Failed to initialize slash transaction")
		}
		txOptsSlash.Nonce = new(big.Int).SetUint64(nonce)
		txOptsSlash.Value = big.NewInt(0)     // in wei
		txOptsSlash.GasLimit = uint64(300000) // in units
		txOptsSlash.GasPrice = gasPrice

		// Make slash transaction
		txSlash, err := contract.Slash(txOptsSlash, stakeSk, stakeID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{"stakeSk": stakeSk, "stakeID": stakeID}).Error("Failed to slash stake")
		} else {
			log.WithField("TX hash", txSlash.Hash()).Info("Successfully submitted slash transaction to Aetherweave contract")
		}

	}

	return nil
}

// Check if a SlashProof is valid
func validateSlashProof(proof *pb.SlashProof, sh_vkey_bytes []byte) error {
	if proof == nil || proof.Record_1 == nil || proof.Record_2 == nil || proof.Slashee == nil {
		return errors.New("incomplete slash proof")
	}

	r1 := proof.Record_1
	r2 := proof.Record_2

	// Check round numbers match
	if r1.RoundNumber != r2.RoundNumber {
		return errors.New("round numbers do not match")
	}

	// Check root hashes differ
	if bytes.Equal(r1.RootHash.Hash, r2.RootHash.Hash) {
		return errors.New("commitment roots are identical")
	}

	// Unmarshal slashee public key
	pubkey, err := crypto.UnmarshalPublicKey(proof.Slashee.Pubkey)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal slashee public key")
	}
	pubkey_bjj, ok := pubkey.(*crypto.BJJPublicKey)
	if !ok {
		return errors.New("public key in slash proof is not a BJJ key")
	}
	pubkey_x, pubkey_y := pubkey_bjj.GetXY()

	// Verify that the share proofs are valid
	for _, commitment := range []*pb.CommitmentRecord{r1, r2} {

		// Unpack proof of stake proof data
		zkp_d, err := ZKPToProofData(commitment.ShareProof)
		if err != nil {
			return errors.Wrap(err, "failed to convert share proof data ZKP protobuf to ProofData")
		}
		// zkp_d := snarktypes.ProofData{}
		// err = json.Unmarshal(commitment.ShareProof, &zkp_d)
		// if err != nil {
		// 	return errors.Wrap(err, "failed to unmarshal share proof data")
		// }

		// Reconstruct public signals from the protobufs what we received
		// Public signals are decimal strings of [ pubkeyX, pubkeyY, commitment_root, epoch, slashshare ]
		commitment_root_bi := new(big.Int).SetBytes(commitment.RootHash.Hash)
		share := new(big.Int).SetBytes(commitment.SlashShare)
		zkp_pub := []string{pubkey_x.String(), pubkey_y.String(), commitment_root_bi.String(), strconv.FormatUint(commitment.RoundNumber, 10), share.String()}
		zkp := snarktypes.ZKProof{Proof: zkp_d, PubSignals: zkp_pub}

		// Check share proof
		err = verifyZKProof(zkp, sh_vkey_bytes)
		if err != nil {
			return errors.Wrap(err, "slash proof verification failed")
		}
	}

	return nil
}

// Fetch latest merkle root from the smart contract
func getSCRoot(contract awContractCaller) (Hash, error) {

	// Call the contract function
	opts := &bind.CallOpts{
		Context: context.Background(),
	}
	root, err := contract.GetRoot(opts)
	if err != nil {
		log.WithError(err).Error("Failed to call AetherWeavePrivate:GetRoot()")
		return Hash{}, err
	}
	return Hash(root), nil
}

// Update the SC root map and return true if a new root has been learned
func (aw *Aetherweave) updateSCRoots() bool {
	// Fetch current root from smart contract
	current_sc_root, err := getSCRoot(aw.contractCaller)
	if err != nil {
		log.WithError(err).Error("Failed to update SC roots")
		return false
	}

	// Get mutex on sc roots and update
	new_root := false
	aw.sc_roots.sc_table_m.Lock()
	defer aw.sc_roots.sc_table_m.Unlock()

	// Check if the fetched root exists in the sc_table
	if _, exists := aw.sc_roots.sc_table[current_sc_root]; !exists {
		new_root = true
	}

	aw.sc_roots.sc_table[current_sc_root] = aw.round_number

	// Find and remove old sc roots from our tables
	var old_sc_roots []Hash
	for root, round := range aw.sc_roots.sc_table {
		if round < aw.round_number-MAX_SC_ROOT_AGE {
			old_sc_roots = append(old_sc_roots, root)
		}
	}
	for _, root := range old_sc_roots {
		delete(aw.sc_roots.sc_table, root)
	}

	return new_root
}

// Check if an sc root is in our sc_roots map and is valid
func (sc *SCRootsTable) scRootValid(sc_root Hash, curr_round_n RoundNumber) bool {
	sc.sc_table_m.RLock()
	defer sc.sc_table_m.RUnlock()
	if round, ok := sc.sc_table[sc_root]; ok && round >= curr_round_n-MAX_SC_ROOT_AGE {
		return true
	}
	return false
}

// Process marshalled form of Aetherweave's PublicKey to get the crypto.Pubkey
// and the corresponding peer.ID / PublicKeyHash
func processMarshalledPubkey(pk_bytes []byte) (crypto.PubKey, peer.ID, error) {

	crypto_pk, err := crypto.UnmarshalPublicKey(pk_bytes)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal public key")
		return nil, "", err
	}
	peerID, err := peer.IDFromPublicKey(crypto_pk)
	if err != nil {
		log.WithError(err).Error("Failed to get peerID from public key")
		return crypto_pk, "", err
	}

	return crypto_pk, peerID, nil
}

// Write NetworkRecord to a .nr file
func writeLocalNetworkRecord(netRec *pb.NetworkRecord, fpath string) error {
	node_net_record_bytes, err := netRec.MarshalSSZ()
	if err != nil {
		return err
	}
	if err := file.WriteFile(fpath, node_net_record_bytes); err != nil {
		return err
	}
	return nil
}

// readLocalNetworkRecords looks for *.rn files in the given folder,
// loads their content and then deserializes it into a list of NetworkRecords
func readLocalNetworkRecords(folderPath string) ([]*pb.NetworkRecord, error) {
	networkRecords := make([]*pb.NetworkRecord, 0)

	// Construct the pattern to find all .rn files in the given folder.
	searchPattern := filepath.Join(folderPath, "*.nr")

	// Find all files matching the pattern.
	files, err := filepath.Glob(searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files in %s: %w", folderPath, err)
	}

	// Iterate over each found file.
	for _, fpath := range files {
		// Read the hex-encoded content from the file
		decodedSSZBytes, err := os.ReadFile(fpath)
		if err != nil {
			log.WithError(err).WithField("fpath", fpath).Warningf("Warning: failed to read file")
			continue
		}

		// Create a new NetworkRecord instance to unmarshal into.
		netRec := &pb.NetworkRecord{}

		// Unmarshal the SSZ bytes into the NetworkRecord
		if err := netRec.UnmarshalSSZ(decodedSSZBytes); err != nil {
			log.WithError(err).WithField("fpath", fpath).Warningf("Warning: failed to unmarshal SSZ")
			continue
		}

		// Add the successfully unmarshaled record to our list.
		networkRecords = append(networkRecords, netRec)
	}

	return networkRecords, nil
}

// Drop-in replacement for listenForNewNodes from the p2p package
func (s *Service) listenForNewNodesAW() {
	const (
		minLogInterval = 1 * time.Minute
	)

	var pollingPeriod = 6 * time.Second

	peersSummary := func(threshold uint) (uint, uint) {
		// Retrieve how many active peers we have.
		activePeers := s.cfg.p2p.Peers().Active()
		activePeerCount := uint(len(activePeers))

		// Compute how many peers we are missing to reach the threshold.
		if activePeerCount >= threshold {
			return activePeerCount, 0
		}

		missingPeerCount := threshold - activePeerCount

		return activePeerCount, missingPeerCount
	}

	lastLogTime := time.Now()
	connectivityTicker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-connectivityTicker.C:
			if s.cfg.p2p.IsPeerAtLimit(false /* inbound */) {
				// Pause the main loop for a period to stop looking
				// for new peers.
				log.Trace("Not looking for peers, at peer limit")
				time.Sleep(pollingPeriod)
				continue
			}

			// Compute the number of new peers we want to dial.
			activePeerCount, missingPeerCount := peersSummary(s.cfg.p2p.GetMaxPeers())

			fields := logrus.Fields{
				"currentPeerCount": activePeerCount,
				"targetPeerCount":  s.cfg.p2p.GetMaxPeers(),
			}

			if missingPeerCount == 0 {
				log.Trace("Not looking for peers, at peer limit")
				time.Sleep(pollingPeriod)
				continue
			}

			if time.Since(lastLogTime) > minLogInterval {
				lastLogTime = time.Now()
				log.WithFields(fields).Debug("Searching for new active peers")
			}

			// Restrict dials if limit is applied.
			if flags.MaxDialIsActive() {
				maxConcurrentDials := uint(flags.Get().MaxConcurrentDials)
				missingPeerCount = min(missingPeerCount, maxConcurrentDials)
			}

			// Get local peerID
			local_aw_peerID, err := peer.IDFromPublicKey(s.aw.node_privkey.GetPublic())
			if err != nil {
				log.WithError(err).Error("Failed to get local aw_peerID")
				return
			}

			// Sample table for new peers
			_, sampledPeers, err := s.aw.table.samplePublicKeys(missingPeerCount, !DUAL_TABLES, map[PublicKeyHash]bool{PublicKeyHash(local_aw_peerID): true})
			if err != nil {
				log.WithError(err).Warning("Failed to sample table keys. Sleeping.")
				time.Sleep(pollingPeriod)
				continue
			}

			log.WithFields(logrus.Fields{"#sampledPeers": len(sampledPeers)}).Info("listenForNewNodesAw: sampled peers table")
			wg := new(sync.WaitGroup)
			for _, aw_peerID := range sampledPeers {
				peer_record, ok := s.aw.table.getRecord(aw_peerID)
				if !ok {
					log.Warn("Unexpected missing record")
					continue
				}

				// Unmarshal peer multiaddr
				peer_multiaddr, err := ma.NewMultiaddrBytes(peer_record.GetNetRecord().GetMultiaddr().GetMultiaddr())
				if err != nil {
					log.WithFields(logrus.Fields{"function": "listenForNewNodesAW", "aw_peerID": aw_peerID}).WithError(err).Error("Could not unmarshall peer addrinfo")
					continue
				}

				// Get the native peerID from the multiaddr
				peerID, err := GetPeerIDFromMultiaddr(peer_multiaddr)
				if err != nil {
					log.WithField("multiaddr", peer_multiaddr).Error("Failed to GetPeerIDFromMultiaddr()")
					continue
				}

				// Ignore nodes that are already connected.
				if s.cfg.p2p.Host().Network().Connectedness(peerID) == network.Connected {
					log.WithFields(logrus.Fields{"function": "listenForNewNodesAW", "peerID": peerID}).Info("Skipping connection to already connected peer")
					continue
				}

				// Add peer to peer handler
				s.cfg.p2p.Peers().Add(nil, peer.ID(peerID), peer_multiaddr, network.DirUnknown)

				// Make sure that peer is not dialed too often, for each connection attempt there's a backoff period.
				s.cfg.p2p.Peers().RandomizeBackOff(peer.ID(peerID))
				wg.Add(1)
				go func(info *peer.AddrInfo) {
					if err := s.connectWithPeer(s.ctx, *info); err != nil {
						log.WithError(err).Tracef("Could not connect with peer %s", info.String())
					}
					wg.Done()
				}(&peer.AddrInfo{ID: peer.ID(peerID), Addrs: []ma.Multiaddr{peer_multiaddr}})
			}
			wg.Wait()
		default:
			time.Sleep(pollingPeriod)
		}
	}
}

// WithAetherweaveKeys adds an ECDSA private key to Aetherweave for deposits and slashes.
// Used as an Option during sync service construction
func WithAetherweaveKeys(awkey string) Option {

	var eth_privkey *ecdsa.PrivateKey
	var eth_pubkey *ecdsa.PublicKey
	var err error

	if len(awkey) > 0 {
		log.WithField("privkey", awkey).Info("ECDSA private key passed via CLI")
		// Initialize Ethereum wallet keys. Used for stake deposit and slashing
		eth_privkey, eth_pubkey, err = ethKeys(awkey)
		if err != nil {
			log.WithError(err).Error("Failed to parse Eth wallet keys")
			return func(s *Service) error {
				log.Error("Failed to parse Eth wallet keys. Bad option.")
				s.aw.eth_privkey = nil
				s.aw.eth_pubkey = nil
				return errors.New("Failed to parse Eth wallet keys. Bad option.")
			}
		}
	} else {
		log.Info("No ECDSA private key passed via CLI - generating one")
		eth_privkey, err = ecdsa.GenerateKey(elliptic.P256(), gorand.Reader)
		eth_pubkey = &eth_privkey.PublicKey
		if err != nil {
			log.WithError(err).Error("Failed to generate Eth wallet keys")
			return func(s *Service) error {
				log.Error("Failed to generate Eth wallet keys. Bad option.")
				s.aw.eth_privkey = nil
				s.aw.eth_pubkey = nil
				return errors.New("Failed to parse Eth wallet keys. Bad option.")
			}
		}
	}

	log.WithField("privkey", fmt.Sprintf("%x", eth_privkey.D)).Info("ECDSA private key used for Aetherweave contract")

	return func(s *Service) error {
		s.aw.eth_privkey = eth_privkey
		s.aw.eth_pubkey = eth_pubkey
		return nil
	}
}

// Get the peerID included in a multiaddr
func GetPeerIDFromMultiaddr(peer_multiaddr ma.Multiaddr) (peer.ID, error) {
	// Get the protocol ID for "p2p" in the multiaddr to get the native peerID
	var protocolCode int = -1
	for _, protocol := range peer_multiaddr.Protocols() {
		if protocol.Name == "p2p" {
			protocolCode = protocol.Code
		}
	}
	if protocolCode == -1 {
		log.WithField("multiaddr", peer_multiaddr).Error("Couldn't find p2p protocol to retrieve native peerID")
		return peer.ID(""), errors.New("Couldn't find p2p protocol to retrieve native peerID")
	}

	// Get native peerID
	peerID_s, err := peer_multiaddr.ValueForProtocol(protocolCode)
	if err != nil {
		log.WithField("multiaddr", peer_multiaddr).WithError(err).Error("Failed to retrieve peerID from multiaddr")
		return peer.ID(""), errors.New("Failed to retrieve peerID from multiaddr")
	}
	return peer.Decode(peerID_s)
}

// Utility function that logs the PeerStore status
func (s *Service) logPeerStoreStatus() {
	peerStoreIDs := s.cfg.p2p.Peers().All()
	log.WithField("peerstore", peerStoreIDs).Info("peerstore status")
	for _, peerStoreID := range peerStoreIDs {
		address, err := s.cfg.p2p.Peers().Address(peerStoreID)
		if err != nil {
			log.WithField("peerID", peerStoreID).WithError(err).Error("No address in peer store")
		} else {
			log.WithFields(logrus.Fields{
				"peerID":    peerStoreID,
				"addresses": address}).Info("peerstore ID")
		}
	}
}
