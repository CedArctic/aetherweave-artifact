package sync

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	awc "github.com/OffchainLabs/prysm/v6/beacon-chain/sync/awcontract"
	pb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/iden3/go-rapidsnark/prover"
	"github.com/iden3/go-rapidsnark/types"
	"github.com/iden3/go-rapidsnark/verifier"
	"github.com/iden3/go-rapidsnark/witness/v2"
)

// Paths to Circom files as placed in the docker image

// Stake Proof circuit, prover key and verifier key
const ST_CIRC = "stake_proof.wasm"
const ST_PKEY = "stake_proof_0001.zkey"
const ST_VKEY = "stake_proof_verification_key.json"

// Share Proof circuit, prover key and verifier key
const SH_CIRC = "share_proof.wasm"
const SH_PKEY = "share_proof_0001.zkey"
const SH_VKEY = "share_proof_verification_key.json"

type StakeProofInputData struct {
	Secret              string   `json:"secret"`
	NetPK               []string `json:"netPK"`
	StakeID             string   `json:"stakeID"`
	MerkleRoot          string   `json:"merkle_root"`
	MerkleProofSiblings []string `json:"merkle_proof_siblings"`
	MerkleProofKey      string   `json:"merkle_proof_key"`
	MerkleProofValue    string   `json:"merkle_proof_value"`
}

type ShareProofInputData struct {
	Secret         string   `json:"secret"`
	NetPK          []string `json:"netPK"`
	StakeID        string   `json:"stakeID"`
	CommitmentRoot string   `json:"commitment_root"`
	SlashShare     string   `json:"slashShare"`
	Epoch          string   `json:"epoch"`
}

// Create input.json for StakeProof and serialize it
func prepareStakeProofInput(secret crypto.PrivKey, stakeID *big.Int, proof *awc.SparseMerkleTreeProof) ([]byte, error) {

	// Get BJJ private key
	bjj_privkey, ok := secret.(*crypto.BJJPrivateKey)
	if !ok {
		return nil, errors.New("Private key is not a Babyjubjub key")
	}

	// Assert that the pubkey is a BabyJubJub key
	bjj_netpk, ok := secret.GetPublic().(*crypto.BJJPublicKey)
	if !ok {
		return nil, errors.New("Public key is not a Babyjubjub key")
	}

	// Get public key bytes
	netpk_X, netpk_Y := bjj_netpk.GetXY()

	// Prepare the StakeProofInputData struct
	inputData := StakeProofInputData{
		Secret:     bjj_privkey.SkToBigInt().String(),
		NetPK:      []string{netpk_X.String(), netpk_Y.String()},
		StakeID:    stakeID.String(),
		MerkleRoot: fmt.Sprintf("0x%x", new(big.Int).SetBytes(proof.Root[:])),
		MerkleProofSiblings: func(siblings [][32]byte) []string {
			hexSiblings := make([]string, len(siblings))
			for i, s := range siblings {
				hexSiblings[i] = fmt.Sprintf("0x%x", new(big.Int).SetBytes(s[:]))
			}
			return hexSiblings
		}(proof.Siblings),
		MerkleProofKey:   fmt.Sprintf("0x%x", new(big.Int).SetBytes(proof.Key[:])),
		MerkleProofValue: fmt.Sprintf("0x%x", new(big.Int).SetBytes(proof.Value[:])),
	}

	// Marshal the struct into a pretty-printed JSON byte slice
	jsonData, err := json.MarshalIndent(inputData, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal StakeProofInputData json")
	}

	return jsonData, nil
}

// Create input.json for StakeProof and serialize it
func prepareShareProofInput(secret crypto.PrivKey, stakeID *big.Int, commitmentRoot Hash, slashShare *big.Int, roundNumber RoundNumber) ([]byte, error) {

	// Get BJJ private key
	bjj_privkey, ok := secret.(*crypto.BJJPrivateKey)
	if !ok {
		return nil, errors.New("Private key is not a Babyjubjub key")
	}

	// Assert that the pubkey is a BabyJubJub key
	bjj_netpk, ok := secret.GetPublic().(*crypto.BJJPublicKey)
	if !ok {
		return nil, errors.New("Public key is not a Babyjubjub key")
	}

	// Get public key bytes
	netpk_X, netpk_Y := bjj_netpk.GetXY()

	// Convert commitmentRoot to big.Int string
	commitmentRoot_bi := new(big.Int).SetBytes(commitmentRoot[:])

	// Prepare the StakeProofInputData struct
	inputData := ShareProofInputData{
		Secret:         bjj_privkey.SkToBigInt().String(),
		NetPK:          []string{netpk_X.String(), netpk_Y.String()},
		StakeID:        stakeID.String(),
		CommitmentRoot: commitmentRoot_bi.String(),
		SlashShare:     slashShare.String(),
		Epoch:          strconv.FormatUint(uint64(roundNumber), 10),
	}

	// Marshal the struct into a pretty-printed JSON byte slice
	jsonData, err := json.MarshalIndent(inputData, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal StakeProofInputData json")
	}

	return jsonData, nil
}

// Generate ZK Proof. Used for stake proofs and share proofs.
func generateZKProof(jsonDataInput []byte, wcalc witness.Calculator, zkeyBytes []byte) (*types.ZKProof, error) {

	// Parse inputs
	inputs_s := map[string]interface{}{}
	if err := json.Unmarshal(jsonDataInput, &inputs_s); err != nil {
		log.WithError(err).Error("Failed to parse inputs while generating stake proof")
		return nil, err
	}

	// Convert to *big.Int or []*big.Int values
	inputs := map[string]interface{}{}
	for k, v := range inputs_s {
		base := 10
		v_s, is_string := v.(string)

		// If value is a big.Int string
		if is_string {
			// Account for hex string inputs
			if len(v_s) > 2 && v_s[:2] == "0x" {
				base = 16
				v_s = v_s[2:]
			}
			// Convert input string to big.Int
			var ok bool
			inputs[k], ok = new(big.Int).SetString(v_s, base)
			if !ok {
				return nil, errors.New(fmt.Sprintf("Failed to convert string to base 10 big.Int: %v", v_s))
			}
		} else if v_as, is_array := v.([]interface{}); is_array { // If value is a list of big.Int strings
			v_abi := make([]*big.Int, len(v_as))
			// Iterate over array items. They should be strings
			for idx, val := range v_as {
				val_s, ok := val.(string)
				if !ok {
					return nil, errors.New(fmt.Sprintf("Expected string array value. Got: %v", val))
				}
				// Account for hex inputs
				if val_s[:2] == "0x" {
					base = 16
					val_s = val_s[2:]
				}
				// Convert input string to big.Int
				var ok2 bool
				v_abi[idx], ok2 = new(big.Int).SetString(val_s, base)
				if !ok2 {
					return nil, errors.New(fmt.Sprintf("Failed to convert string to base 10 big.Int: %v", val))
				}
			}
			inputs[k] = v_abi
		} else { // This should not happen
			return nil, errors.New(fmt.Sprintf("Failed to parse zk inputs: %v", inputs_s))
		}
	}

	// log.WithFields(logrus.Fields{"function": "CalculateWTNSBin", "RSS": getRSS()}).Info("RSS Before")
	wtns, err := wcalc.CalculateWTNSBin(inputs, true) // true = sanity-check constraints
	if err != nil {
		log.WithError(err).Error("Failed to calculate witnesses")
		return nil, err
	}
	// log.WithFields(logrus.Fields{"function": "CalculateWTNSBin", "RSS": getRSS()}).Info("RSS After")

	// [Debug] Write witness to disk
	// wtnsFpath := path.Join(cmd.DefaultDataDir(), "witness_temp.wtns")
	// _ = os.WriteFile(wtnsFpath, wtns, 0644)

	// Generate Groth16 proof using RapidSNARK (Circom-compatible)
	// log.WithFields(logrus.Fields{"function": "Groth16Prover", "RSS": getRSS()}).Info("RSS Before")
	prv, err := prover.Groth16Prover(zkeyBytes, wtns)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	// log.WithFields(logrus.Fields{"function": "Groth16Prover", "RSS": getRSS()}).Info("RSS After")

	// Save Circom-style proof + public signals. Can be used for snarkjs verify
	// saveJSON("proof.json", prv.Proof)
	// saveJSON("public_signals.json", prv.PubSignals)

	return prv, nil
}

// Verify a Stake Proof. Used for stake proofs and share proofs.
func verifyZKProof(zkProof types.ZKProof, vkeyBytes []byte) error {
	// log.WithFields(logrus.Fields{"function": "VerifyGroth16", "RSS": getRSS()}).Info("RSS Before")
	// defer log.WithFields(logrus.Fields{"function": "VerifyGroth16", "RSS": getRSS()}).Info("RSS After")
	return verifier.VerifyGroth16(zkProof, vkeyBytes)
}

// func saveJSON(path string, v any) {
// 	b, _ := json.MarshalIndent(v, "", "  ")
// 	_ = os.WriteFile(path, b, 0644)
// }

// stringToBytes32 converts a base-10 string into a 32-byte big-endian slice.
func stringToBytes32(s string) ([]byte, error) {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, errors.New("invalid integer string")
	}

	if n.Sign() < 0 {
		return nil, errors.New("negative integers not supported")
	}

	b := n.Bytes()
	if len(b) > 32 {
		return nil, errors.New("integer exceeds 256 bits")
	}

	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out, nil
}

// bytes32ToString converts a 32-byte big-endian slice into a base-10 string.
func bytes32ToString(b []byte) string {
	n := new(big.Int).SetBytes(b)
	return n.String()
}

func ProofDataToZKP(p *types.ProofData) (*pb.ZKP, error) {
	if len(p.A) != 3 {
		return nil, errors.New("pi_a must have exactly 3 elements")
	}
	if len(p.C) != 3 {
		return nil, errors.New("pi_c must have exactly 3 elements")
	}
	if len(p.B) != 3 || len(p.B[0]) != 2 || len(p.B[1]) != 2 || len(p.B[2]) != 2 {
		return nil, errors.New("pi_b must be a 2x2 matrix")
	}

	a0, err := stringToBytes32(p.A[0])
	if err != nil {
		return nil, err
	}
	a1, err := stringToBytes32(p.A[1])
	if err != nil {
		return nil, err
	}
	a2, err := stringToBytes32(p.A[2])
	if err != nil {
		return nil, err
	}

	c0, err := stringToBytes32(p.C[0])
	if err != nil {
		return nil, err
	}
	c1, err := stringToBytes32(p.C[1])
	if err != nil {
		return nil, err
	}
	c2, err := stringToBytes32(p.C[2])
	if err != nil {
		return nil, err
	}

	b00, err := stringToBytes32(p.B[0][0])
	if err != nil {
		return nil, err
	}
	b01, err := stringToBytes32(p.B[0][1])
	if err != nil {
		return nil, err
	}
	b10, err := stringToBytes32(p.B[1][0])
	if err != nil {
		return nil, err
	}
	b11, err := stringToBytes32(p.B[1][1])
	if err != nil {
		return nil, err
	}
	b20, err := stringToBytes32(p.B[2][0])
	if err != nil {
		return nil, err
	}
	b21, err := stringToBytes32(p.B[2][1])
	if err != nil {
		return nil, err
	}

	return &pb.ZKP{
		PiA: &pb.PI_A{
			A: a0,
			B: a1,
			C: a2,
		},
		PiB: &pb.PI_B{
			Points: []*pb.PI_B_Point{
				{A: b00, B: b01},
				{A: b10, B: b11},
				{A: b20, B: b21},
			},
		},
		PiC: &pb.PI_C{
			A: c0,
			B: c1,
			C: c2,
		},
	}, nil
}

func ZKPToProofData(z *pb.ZKP) (*types.ProofData, error) {
	if z == nil || z.PiA == nil || z.PiB == nil || z.PiC == nil {
		return nil, errors.New("zkp or submessages are nil")
	}
	if len(z.PiB.Points) != 3 {
		return nil, errors.New(fmt.Sprintf("pi_b must have exactly 3 points. Pi_B: %v", z.PiB.Points))
	}

	return &types.ProofData{
		A: []string{
			bytes32ToString(z.PiA.A),
			bytes32ToString(z.PiA.B),
			bytes32ToString(z.PiA.C),
		},
		B: [][]string{
			{
				bytes32ToString(z.PiB.Points[0].A),
				bytes32ToString(z.PiB.Points[0].B),
			},
			{
				bytes32ToString(z.PiB.Points[1].A),
				bytes32ToString(z.PiB.Points[1].B),
			},
			{
				bytes32ToString(z.PiB.Points[2].A),
				bytes32ToString(z.PiB.Points[2].B),
			},
		},
		C: []string{
			bytes32ToString(z.PiC.A),
			bytes32ToString(z.PiC.B),
			bytes32ToString(z.PiC.C),
		},
		Protocol: "groth16",
	}, nil
}

// Get allocated memory from the OS
func getRSS() string {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "unknown"
	}
	// Look for VmRSS line
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			return line
		}
	}
	return "unknown"
}
