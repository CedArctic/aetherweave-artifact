# Script to generate zk proof inputs using the solidity implementation
# of the merkle tree.
# Run with, e.g.: ape run generate
import json
import os
from typing import Any

import pytest
import snark
import utils
from utils import (
    AWAccount,
    compute_share,
    save_input,
    share_proof_input_data,
    stake_proof_input_data,
)

ST_PATH = "./output/stake_proof"
ST_PROVING_KEY = "./trusted-setup/stake_proof_0001.zkey"
ST_VERIFICATION_KEY = "./trusted-setup/stake_proof_verification_key.json"
SL_PATH = "./output/share_proof"
SL_PROVING_KEY = "./trusted-setup/share_proof_0001.zkey"
SL_VERIFICATION_KEY = "./trusted-setup/share_proof_verification_key.json"


def test_stake_proof_verifier(
    aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    # deposit funds to staking contract
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed
    receipt = aetherweave.deposit(bob.stakeID, sender=bob.eth, value="1 Eth")
    assert not receipt.failed

    # Generate the merkle proof using the Solidity implementation
    root = aetherweave.getRoot()
    print(f"Merkle root: {utils.pretty_hex(root)}")
    proof = aetherweave.getProof(alice.stakeID, sender=alice.eth)
    utils.pretty_print_proof(proof)

    # make input.json for witness generation
    input_data = stake_proof_input_data(alice, root, proof)
    save_input(input_data, ST_PATH)

    # Generate the witness and proof
    snark.generate_witness(
        f"{ST_PATH}/input.json",
        f"{ST_PATH}/stake_proof_js/stake_proof.wasm",
        ST_PATH,
    )
    snark.generate_proof(ST_PROVING_KEY, f"{ST_PATH}/witness.wtns", ST_PATH)

    # test proof verification
    snark.verify_proof(
        ST_VERIFICATION_KEY,
        f"{ST_PATH}/public.json",
        f"{ST_PATH}/proof.json",
    )


def test_share_proof_verifier(alice: AWAccount) -> None:
    # random made up values for testing, these are validated in the protocol
    commitment_root = 462176416242184
    epoch = 1

    # compute share
    share = compute_share(alice.secret, epoch, commitment_root)

    # make input.json
    input_data = share_proof_input_data(alice, commitment_root, epoch, share)
    save_input(input_data, SL_PATH)

    # Generate the witness and proof
    snark.generate_witness(
        f"{SL_PATH}/input.json",
        f"{SL_PATH}/share_proof_js/share_proof.wasm",
        SL_PATH,
    )
    snark.generate_proof(SL_PROVING_KEY, f"{SL_PATH}/witness.wtns", SL_PATH)

    # test proof verification
    snark.verify_proof(
        SL_VERIFICATION_KEY,
        f"{SL_PATH}/public.json",
        f"{SL_PATH}/proof.json",
    )


def apply_bob_value(field: str, input: dict[str, Any], bob: AWAccount) -> None:
    if field == "netPK":
        input["netPK"] = [str(x) for x in bob.netPK]
    elif field == "secret":
        input["secret"] = str(bob.secret)
    elif field == "stakeID":
        input["stakeID"] = str(bob.stakeID)
    else:
        raise ValueError(f"Unknown field: {field}")


@pytest.mark.parametrize("field", ["netPK", "secret", "stakeID"])
def test_stake_proof_rejects_wrong_fields(
    aetherweave: Any, alice: AWAccount, bob: AWAccount, field: str
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed

    root = aetherweave.getRoot()
    proof = aetherweave.getProof(alice.stakeID, sender=alice.eth)
    input_data = stake_proof_input_data(alice, root, proof)

    apply_bob_value(field, input_data, bob)
    save_input(input_data, ST_PATH)

    with pytest.raises(snark.SnarkError):
        snark.generate_witness(
            f"{ST_PATH}/input.json",
            f"{ST_PATH}/stake_proof_js/stake_proof.wasm",
            ST_PATH,
        )


@pytest.mark.parametrize("field", ["netPK", "secret", "stakeID"])
def test_share_proof_rejects_wrong_fields(
    aetherweave: Any, alice: AWAccount, bob: AWAccount, field: str
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed

    share = compute_share(bob.secret, 1, 123456789)
    input_data = share_proof_input_data(
        alice, 124124124124124, 4365346346, share
    )

    apply_bob_value(field, input_data, bob)
    save_input(input_data, SL_PATH)

    with pytest.raises(snark.SnarkError):
        snark.generate_witness(
            f"{SL_PATH}/input.json",
            f"{SL_PATH}/share_proof_js/share_proof.wasm",
            SL_PATH,
        )


def test_secret_size_limit(aetherweave: Any, alice: AWAccount) -> None:
    # secret too large

    secret = 2**253
    stake_secret = utils.poseidon(secret)
    stake_id = utils.poseidon(stake_secret)
    net_pk = utils.babyjub_public_key(secret)

    # deposit funds to staking contract
    receipt = aetherweave.deposit(stake_id, sender=alice.eth, value="1 Eth")
    assert not receipt.failed

    root = aetherweave.getRoot()
    proof = aetherweave.getProof(stake_id, sender=alice.eth)
    input_data = stake_proof_input_data(
        AWAccount(
            eth=alice.eth,
            stakeID=stake_id,
            secret=secret,
            stakeSecret=stake_secret,
            netPK=net_pk,
        ),
        root,
        proof,
    )
    save_input(input_data, ST_PATH)

    with pytest.raises(snark.SnarkError):
        snark.generate_witness(
            f"{ST_PATH}/input.json",
            f"{ST_PATH}/stake_proof_js/stake_proof.wasm",
            ST_PATH,
        )


def test_stake_proof_public_encoding(
    aetherweave: Any, alice: AWAccount
) -> None:
    # deposit funds to staking contract
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed

    root = aetherweave.getRoot()
    proof = aetherweave.getProof(alice.stakeID, sender=alice.eth)
    input_data = stake_proof_input_data(alice, root, proof)
    save_input(input_data, ST_PATH)

    # Generate the witness and proof
    snark.generate_witness(
        f"{ST_PATH}/input.json",
        f"{ST_PATH}/stake_proof_js/stake_proof.wasm",
        ST_PATH,
    )
    snark.generate_proof(ST_PROVING_KEY, f"{ST_PATH}/witness.wtns", ST_PATH)

    # test public encoding
    with open(f"{ST_PATH}/public.json", "r") as f:
        public_data = json.load(f)

    # test merkle_root, netPK in public.json
    assert public_data[0] == str(alice.netPK[0]), "NetPK[0] mismatch"
    assert public_data[1] == str(alice.netPK[1]), "NetPK[1] mismatch"
    assert public_data[2] == str(
        int.from_bytes(root, byteorder="big")
    ), "Merkle root mismatch"


def test_share_proof_public_encoding(
    aetherweave: Any, alice: AWAccount
) -> None:
    commitment_root = 5125342151
    epoch = 14351

    share = compute_share(alice.secret, epoch, commitment_root)
    input_data = share_proof_input_data(alice, commitment_root, epoch, share)
    save_input(input_data, SL_PATH)

    snark.generate_witness(
        f"{SL_PATH}/input.json",
        f"{SL_PATH}/share_proof_js/share_proof.wasm",
        SL_PATH,
    )
    snark.generate_proof(SL_PROVING_KEY, f"{SL_PATH}/witness.wtns", SL_PATH)

    # test public encoding
    with open(f"{SL_PATH}/public.json", "r") as f:
        public_data = json.load(f)

    # test commitment_root, netPK, epoch, share in public.json
    assert public_data[0] == str(alice.netPK[0]), "NetPK[0] mismatch"
    assert public_data[1] == str(alice.netPK[1]), "NetPK[1] mismatch"
    assert public_data[2] == str(commitment_root), "Commitment root mismatch"
    assert public_data[3] == str(epoch), "Epoch mismatch"
    assert public_data[4] == str(share), "Share mismatch"
