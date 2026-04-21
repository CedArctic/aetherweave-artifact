import json
import os
import random
import subprocess
from collections import namedtuple
from typing import Any

import ape.types.signatures
import eth_account
import eth_typing
import web3
from circomlibpy.poseidon import PoseidonHash  # type: ignore
from hexbytes import HexBytes

EthereumAddress = eth_typing.ChecksumAddress
ZERO_ADDRESS = "0x0000000000000000000000000000000000000000"

BABY = "./scripts/babyjub.js"
DAY = 60 * 60 * 24
HOUR = 60 * 60
PRIME = 21888242871839275222246405745257275088548364400416034343698204186575808495617
ORDER = (
    2736030358979909402780800718157159386076813972158567259200215660948447373041
)

AWAccount = namedtuple(
    "AWAccount", ["eth", "stakeID", "secret", "stakeSecret", "netPK"]
)


def to_32byte_hex(val: bytes) -> str:
    return web3.Web3.to_hex(web3.Web3.to_bytes(val).rjust(32, b"\0"))


def get_message_hash(data: list[Any], types: list[str]) -> HexBytes:
    return HexBytes(web3.Web3.solidity_keccak(types, data))


# def eth_sign(account: Any, num: int, key: HexBytes) -> Any:
def eth_sign(account: Any, data: list[Any], types: list[str]) -> Any:
    """returns a new version of the given state message,
    signed by the given private key. The signature is added to the new message.
    """

    message_hash = get_message_hash(data, types)
    encoded_msg = eth_account.messages.encode_defunct(message_hash)
    sig = account.sign_message(encoded_msg)
    return sig


def eth_validate_signature(
    data: list[Any], types: list[str], sig: Any, pk: EthereumAddress
) -> bool:
    """validates the signature of the channel state message"""
    message_hash = get_message_hash(data, types)
    encoded_msg = eth_account.messages.encode_defunct(message_hash)
    return ape.types.signatures.recover_signer(encoded_msg, sig) == pk


def pretty_hex(value: bytes) -> str:
    """Format a hex value for better readability."""
    return "0x" + value.hex()[:4] + ".." + value.hex()[-4:]


def pretty_print_proof(proof: Any) -> None:
    print("Generated proof:")
    print("  Root:           ", pretty_hex(proof.root))
    print("  Key:            ", pretty_hex(proof.key))
    print("  Value:          ", pretty_hex(proof.value))
    print(
        "  Siblings:       ",
        [pretty_hex(sibling) for sibling in proof.siblings],
    )
    print("  Existence:      ", proof.existence)
    print("  Aux Existence:  ", proof.auxExistence)
    print("  Aux Key:        ", proof.auxKey.hex())
    print("  Aux Value:      ", proof.auxValue.hex())


# poseidon hash with variable number of arguments
def poseidon(*args: int) -> int:
    if len(args) > 4:
        raise ValueError("Too many arguments")
    ps = PoseidonHash()
    x: int = ps.hash(len(args), list(args))
    return x


def babyjub_public_key(sk: int) -> tuple[int, int]:
    payload = {"sk": str(int(sk))}
    p = subprocess.run(
        ["node", BABY],
        input=json.dumps(payload).encode(),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
    )
    response = json.loads(p.stdout.decode())
    if not response.get("ok"):
        raise RuntimeError(response.get("error", "Unknown error"))
    return int(response["x"]), int(response["y"])


def stake_proof_input_data(
    alice: AWAccount, root: bytes, proof: Any
) -> dict[str, Any]:
    input_data = {
        "secret": str(alice.secret),
        "netPK": [str(x) for x in alice.netPK],
        "stakeID": str(alice.stakeID),
        "merkle_root": "0x" + root.hex(),
        "merkle_proof_siblings": [
            "0x" + sibling.hex() for sibling in proof.siblings
        ],
        "merkle_proof_key": "0x" + proof.key.hex(),
        "merkle_proof_value": "0x" + proof.value.hex(),
    }
    return input_data


def share_proof_input_data(
    alice: AWAccount, commitment_root: int, epoch: int, share: int
) -> dict[str, Any]:
    input_data = {
        "secret": str(alice.secret),
        "netPK": [str(x) for x in alice.netPK],
        "stakeID": str(alice.stakeID),
        "commitment_root": str(commitment_root),
        "slashShare": str(share),
        "epoch": str(epoch),
    }
    return input_data


def save_input(input_data: dict[str, Any], folder: str) -> None:
    """Saves the input data to input.json file."""
    os.makedirs(folder, exist_ok=True)
    with open(os.path.join(folder, "input.json"), "w") as f:
        json.dump(input_data, f, indent=4)


def compute_share(secret: int, epoch: int, commitment_root: int) -> int:
    a = poseidon(secret, epoch, 448612363379)
    stake_secret = poseidon(secret)
    return (a * commitment_root + stake_secret) % PRIME
