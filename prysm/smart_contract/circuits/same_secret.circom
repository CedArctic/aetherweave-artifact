pragma circom 2.1.8;

//include "circomlib/circuits/escalarmulfix.circom";
include "./lib/escalarmulfix.circom";
include "./lib/solarity/hasher/poseidon/poseidon.circom";

// prove that p1 and p2 are both derived from the same secret scalar s
// P1 = s·G, P2 = s·H, where G and H are fixed bases
// P1 is hidden, P2 is public
template SameSecret() {

    /* private inputs */
    signal input secret;
    signal input stakeID;

    /* public inputs */
    signal input netPK[2];

    /* fixed bases */
    var G[2] = [
        5299619240641551281634865583518297030282874472190772894086521144482721001553,
        16950150798460657717958625567821834550301663161624707787222815936182638968203
    ];

    /* verify secret -> stakeSecret */
    component hash = Poseidon(1);
    hash.in[0] <== secret;
    hash.dummy <== 0;
    signal stakeSecret;
    stakeSecret <== hash.out;

    log("secret: ", secret);
    log("stakeSecret: ", stakeSecret);
    log("stakeSecret[computed]: ", hash.out);

    /* verify stakeSecret -> stakeID */
    component stakeHash = Poseidon(1);
    stakeHash.in[0] <== stakeSecret;
    stakeHash.dummy <== 0;

    log("stakeID: ", stakeID);
    log("stakeID[computed]: ", stakeHash.out);
    stakeID === stakeHash.out;

    /* verify secret -> netPK */
    component sBits = Num2Bits(253);
    sBits.in <== secret;
    component mulG = EscalarMulFix(253, G);
    for (var i = 0; i < 253; i++) {
        mulG.e[i] <== sBits.out[i];
    }
    netPK === mulG.out;
}