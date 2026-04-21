pragma circom 2.1.8;

include "./lib/solarity/data-structures/SparseMerkleTree.circom";
include "./lib/solarity/hasher/poseidon/poseidon.circom";
include "./same_secret.circom";

template ShareProof(DEPTH) {

    signal input secret;
    signal input stakeID;

    /* public inputs */
    signal input netPK[2];
    signal input commitment_root;
    signal input epoch;
    signal input slashShare;

    /* verify keys and ids */
    component sameSecret = SameSecret();
    sameSecret.secret <== secret;
    sameSecret.stakeID <== stakeID;
    sameSecret.netPK <== netPK;

    /* compute stakeSecret */
    component hash1 = Poseidon(1);
    hash1.in[0] <== secret;
    hash1.dummy <== 0;
    signal stakeSecret;
    stakeSecret <== hash1.out;

    /* compute slash share */
    component hash2 = Poseidon(3);
    hash2.in[0] <== secret;
    hash2.in[1] <== epoch;
    hash2.in[2] <== 448612363379; // int representing "slash"
    hash2.dummy <== 0;
    signal a;
    a <== hash2.out;
    slashShare === a * commitment_root + stakeSecret;

}

component main { public [commitment_root, netPK, epoch, slashShare] } = ShareProof(16);
