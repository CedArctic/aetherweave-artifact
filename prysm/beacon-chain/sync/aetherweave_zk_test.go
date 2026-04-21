package sync

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	awc "github.com/OffchainLabs/prysm/v6/beacon-chain/sync/awcontract"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"

	"github.com/iden3/go-iden3-crypto/v2/babyjub"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
	"github.com/iden3/go-rapidsnark/types"
	"github.com/iden3/go-rapidsnark/verifier"
	"github.com/iden3/go-rapidsnark/witness/v2"
	"github.com/iden3/go-rapidsnark/witness/wasmer"
)

func TestPrepareStakeProofInput(t *testing.T) {
	// Generate BabyJubJub keypair
	priv, pub, err := crypto.GenerateBJJKeyPair(rand.Reader)
	require.NoError(t, err)
	require.NotNil(t, pub)

	stakeID := big.NewInt(12345)

	// Create dummy SparseMerkleTreeProof
	proof := &awc.SparseMerkleTreeProof{
		Root:     [32]byte{0xAA},
		Siblings: [][32]byte{{0xBB}, {0xCC}},
		Key:      [32]byte{0xDD},
		Value:    [32]byte{0xEE},
	}

	// Call function under test
	data, err := prepareStakeProofInput(priv, stakeID, proof)
	require.NoError(t, err)

	// Parse back JSON
	var parsed StakeProofInputData
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Check values
	require.Equal(t, stakeID.String(), parsed.StakeID)
	require.Equal(t, "0xaa00000000000000000000000000000000000000000000000000000000000000", parsed.MerkleRoot)
	require.Equal(t, "0xdd00000000000000000000000000000000000000000000000000000000000000", parsed.MerkleProofKey)
	require.Equal(t, "0xee00000000000000000000000000000000000000000000000000000000000000", parsed.MerkleProofValue)

	// Siblings should be hex-encoded with 0x prefix
	require.Len(t, parsed.MerkleProofSiblings, 2)
	require.Equal(t, "0xbb00000000000000000000000000000000000000000000000000000000000000", parsed.MerkleProofSiblings[0])
	require.Equal(t, "0xcc00000000000000000000000000000000000000000000000000000000000000", parsed.MerkleProofSiblings[1])

	// Public key should have X and Y coordinates
	require.Len(t, parsed.NetPK, 2)
	require.NotEmpty(t, parsed.NetPK[0])
	require.NotEmpty(t, parsed.NetPK[1])
}

// End to end example that tests preparing, generating, and verifying a Proof of Stake ZK proof
func TestZKProofOfStake(t *testing.T) {
	// Generate a BabyJubJub keypair from a secret in hex format (34=0x22)
	privKey := babyjub.PrivateKey{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22}
	priv, pub, err := crypto.BJJKeyPairFromKey(&privKey)
	require.NoError(t, err)
	require.NotNil(t, pub)
	bjj_privkey, ok := priv.(*crypto.BJJPrivateKey)
	require.True(t, ok)

	// Generate valid stakeSK and stakeID from the priv key
	priv_SkBi := bjj_privkey.SkToBigInt()
	stakeSK, err := poseidon.Hash([]*big.Int{priv_SkBi})
	require.NoError(t, err)
	stakeID, err := poseidon.Hash([]*big.Int{stakeSK})
	require.NoError(t, err)

	// Prepare a valid sample SparseMerkleTreeProof for the above private key
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

	// Prepare JSON input using prepareStakeProofInput
	inputJSON, err := prepareStakeProofInput(priv, stakeID, proof)
	require.NoError(t, err)

	// Decode and verify JSON inputs
	var inputUnmarshalled StakeProofInputData
	err = json.Unmarshal(inputJSON, &inputUnmarshalled)
	t.Log(inputUnmarshalled)

	// [DEBUG]: Override
	// var inputJSON_loaded StakeProofInputData
	// t.Log(inputJSON_loaded)
	// loadFixture(t, "stake_proof/input.json", &inputJSON_loaded)
	// // Marshal the struct into a pretty-printed JSON byte slice
	// inputJSON, err = json.MarshalIndent(inputJSON_loaded, "", "  ")
	// require.NoError(t, err)

	// Load the wasm and zkey files
	wasmPath := filepath.Join("testdata/zk", ST_CIRC)
	wasmBytes, err := os.ReadFile(wasmPath)
	require.NoError(t, err, "failed to read wasm file")

	zkeyPath := filepath.Join("testdata/zk", ST_PKEY)
	zkeyBytes, err := os.ReadFile(zkeyPath)
	require.NoError(t, err, "failed to read zkey file")

	wcalc, err := witness.NewCalculator(
		wasmBytes,
		witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
	)
	require.NoError(t, err)

	// Call generateZKProof
	zkp, err := generateZKProof(inputJSON, wcalc, zkeyBytes)
	require.NoError(t, err, "generateZKProof should succeed")
	require.NotNil(t, zkp)

	// Validate proof structure
	require.NotNil(t, zkp.Proof)
	require.NotEmpty(t, zkp.PubSignals)

	// Log proof JSON
	zkpJSON, _ := json.MarshalIndent(zkp.Proof, "", "  ")
	t.Logf("Generated Proof: %s", string(zkpJSON))
	t.Logf("Public signals: %v", zkp.PubSignals)

	// Reconstruct public input signals. big.Int decimal strings: [pubX, pubY, merkleRoot]
	bjj_pub, ok := pub.(*crypto.BJJPublicKey)
	require.True(t, ok)
	pubX, pubY := bjj_pub.GetXY()
	merkle_root := new(big.Int).SetBytes(proof.Root[:]).String()
	pub_signals := []string{pubX.String(), pubY.String(), merkle_root}

	// Unmarshall zkp data
	zkp_re := types.ProofData{}
	err = json.Unmarshal(zkpJSON, &zkp_re)

	// Reconstruct proof object
	zkp_r := types.ZKProof{
		Proof:      &zkp_re,
		PubSignals: pub_signals,
	}

	// Verify the proof (equivalent of what verifyZKProof does)
	vkeyPath := filepath.Join("testdata/zk", ST_VKEY)
	vkeyBytes, err := os.ReadFile(vkeyPath)
	require.NoError(t, err, "loading verification key failed")
	start := time.Now()
	err = verifier.VerifyGroth16(zkp_r, vkeyBytes)
	elapsed := time.Since(start)
	t.Logf("Verification time: %v", elapsed)
	require.NoError(t, err, "verification failed")
}

// End to end example that tests preparing, generating, and verifying a share proofs
func TestZKShareProof(t *testing.T) {
	// Generate a BabyJubJub keypair from scalar secret
	// priv, pub, err := crypto.BJJKeyPairFromScalar(new(big.Int).SetInt64(34))
	privKey := babyjub.PrivateKey{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22}
	priv, pub, err := crypto.BJJKeyPairFromKey(&privKey)
	require.NoError(t, err)
	require.NotNil(t, pub)
	bjj_privkey, ok := priv.(*crypto.BJJPrivateKey)
	require.True(t, ok)

	// Generate valid stakeSK and stakeID from the priv key
	priv_SkBi := bjj_privkey.SkToBigInt()
	stakeSK, err := poseidon.Hash([]*big.Int{priv_SkBi})
	require.NoError(t, err)
	stakeID, err := poseidon.Hash([]*big.Int{stakeSK})
	require.NoError(t, err)

	// Valid commitment root, slash share, and epoch
	commitment_root := Hash{}
	comm_root_b := new(big.Int).SetInt64(5125342151).Bytes()
	bytes_w := copy(commitment_root[32-len(comm_root_b):], comm_root_b)
	require.Equal(t, bytes_w, len(comm_root_b))
	round_number := RoundNumber(14351)
	slashShare, err := calculate_share(priv, round_number, commitment_root)
	require.NoError(t, err)

	// Prepare JSON input using prepareStakeProofInput
	inputJSON, err := prepareShareProofInput(priv, stakeID, commitment_root, slashShare, round_number)
	require.NoError(t, err)

	// Decode and verify JSON inputs
	var inputUnmarshalled ShareProofInputData
	err = json.Unmarshal(inputJSON, &inputUnmarshalled)
	t.Log(inputUnmarshalled)

	// Load the wasm and zkey files
	wasmPath := filepath.Join("testdata/zk", SH_CIRC)
	wasmBytes, err := os.ReadFile(wasmPath)
	require.NoError(t, err, "failed to read wasm file")

	zkeyPath := filepath.Join("testdata/zk", SH_PKEY)
	zkeyBytes, err := os.ReadFile(zkeyPath)
	require.NoError(t, err, "failed to read zkey file")

	wcalc, err := witness.NewCalculator(
		wasmBytes,
		witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator),
	)
	require.NoError(t, err)

	// Call generateZKProof
	zkp, err := generateZKProof(inputJSON, wcalc, zkeyBytes)
	require.NoError(t, err, "generateZKProof should succeed")
	require.NotNil(t, zkp)

	// Validate proof structure
	require.NotNil(t, zkp.Proof)
	require.NotEmpty(t, zkp.PubSignals)

	// Log proof JSON
	zkpJSON, _ := json.MarshalIndent(zkp.Proof, "", "  ")
	t.Logf("Generated Proof: %s", string(zkpJSON))
	t.Logf("Public signals: %v", zkp.PubSignals)

	// Reconstruct public input signals. big.Int decimal strings: [pubX, pubY, commitment_root, round_number, slash_share]
	bjj_pub, ok := pub.(*crypto.BJJPublicKey)
	require.True(t, ok)
	pubX, pubY := bjj_pub.GetXY()
	pub_signals := []string{
		pubX.String(),
		pubY.String(),
		new(big.Int).SetBytes(commitment_root[:]).String(),
		strconv.FormatUint(uint64(round_number), 10),
		slashShare.String(),
	}

	// Unmarshall zkp data
	zkp_re := types.ProofData{}
	err = json.Unmarshal(zkpJSON, &zkp_re)

	// Reconstruct proof object
	zkp_r := types.ZKProof{
		Proof:      &zkp_re,
		PubSignals: pub_signals,
	}

	// Verify the proof (equivalent of what verifyZKProof does)
	vkeyPath := filepath.Join("testdata/zk", SH_VKEY)
	vkeyBytes, err := os.ReadFile(vkeyPath)
	start := time.Now()
	err = verifier.VerifyGroth16(zkp_r, vkeyBytes)
	elapsed := time.Since(start)
	t.Logf("Verification time: %v", elapsed)
	require.NoError(t, err, "verification failed")
}
