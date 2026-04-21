pragma circom 2.1.8;

// include "@solarity/circom-lib/data-structures/SparseMerkleTree.circom";
include "./lib/solarity/data-structures/SparseMerkleTree.circom";
include "./lib/solarity/hasher/poseidon/poseidon.circom";
include "./same_secret.circom";

/*
    Given the merkle root, a merkle proof and P2,
    verifies that P1 is in the Merkle tree and both are
    derived using secret s.
*/
template StakeProof(DEPTH) {

    signal input secret;
    signal input stakeID;


    /* Merkle proof inputs */
    signal input merkle_proof_siblings[DEPTH];
    signal input merkle_proof_key;
    signal input merkle_proof_value;

    /* public inputs */
    signal input netPK[2];
    signal input merkle_root;

    /* verify merkle proof */
    component smt = SparseMerkleTree(DEPTH);
    smt.root <== merkle_root;
    smt.dummy <== 0; // dummy input for padding
    smt.isExclusion <== 0;
    smt.siblings <== merkle_proof_siblings;
    smt.key <== merkle_proof_key;
    smt.value <== merkle_proof_value;
    smt.auxKey <== 0;
    smt.auxValue <== 0;
    smt.auxIsEmpty <== 0;

    /* check that netPK is derived from secret */
    component sameSecret = SameSecret();
    sameSecret.secret <== secret;
    sameSecret.stakeID <== stakeID;
    sameSecret.netPK <== netPK;

    /* ensure that the proof is for stakeID */
    merkle_proof_key === stakeID;

    /* check that value is > 0 */
    signal inv;
    inv <-- merkle_proof_value != 0 ? 1/merkle_proof_value : 0;
    1 === inv * merkle_proof_value;
}

component main { public [merkle_root, netPK] } = StakeProof(16);
