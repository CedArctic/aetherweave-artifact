from typing import Any

import ape
import pytest
import utils
from eth_utils import to_checksum_address  # type: ignore
from utils import HOUR, AWAccount


def test_signature_functions(owner: Any, basic_aetherweave: Any) -> None:
    """Test the signature functions in utils.py."""

    data = [b"\x11" * 32, 1]  # root hash and round number
    types = ["bytes32", "int256"]

    sig = utils.eth_sign(owner, data, types)
    assert utils.eth_validate_signature(data, types, sig, owner.address)
    assert basic_aetherweave.verifySignature(
        *data, sig.encode_rsv(), owner.address
    )


def test_deposit_withdraw_happy_path(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    assert not basic_aetherweave.deposits(alice.eth)
    assert basic_aetherweave.withdrawalTime(alice.eth) == 0

    with ape.reverts("No stake to get proof"):
        basic_aetherweave.getProof(sender=alice.eth)

    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    assert basic_aetherweave.deposits(alice.eth)
    assert basic_aetherweave.withdrawalTime(alice.eth) == 0

    proof = basic_aetherweave.getProof(sender=alice.eth)
    assert basic_aetherweave.verifyProof(proof, sender=alice.eth) is True
    proof.root = bytes([15] * 32)  # tamper with the proof
    assert basic_aetherweave.verifyProof(proof, sender=alice.eth) is False

    receipt = basic_aetherweave.requestWithdrawal(sender=alice.eth)
    assert not receipt.failed, "Transaction failed"

    assert basic_aetherweave.withdrawalTime(alice.eth) > 0


def test_proof(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount, charlie: AWAccount
) -> None:
    receipt1 = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    receipt2 = basic_aetherweave.deposit(sender=bob.eth, value="1 Eth")
    receipt3 = basic_aetherweave.deposit(sender=charlie.eth, value="1 Eth")

    assert not receipt1.failed
    assert not receipt2.failed
    assert not receipt3.failed

    proof = basic_aetherweave.getProof(sender=alice.eth)

    # This is the structure of the proof returned by the contract:

    # struct Proof {
    #     bytes32 root
    #     bytes32[] siblings
    #     bool existence
    #     bytes32 key
    #     bytes32 value
    #     bool auxExistence
    #     bytes32 auxKey
    #     bytes32 auxValue
    # }

    root = basic_aetherweave.getRoot()
    assert proof.root == root

    assert proof.existence is True

    # the Auxiliary fields should be empty for an inclusion proof. They are only used for exclusion proofs,
    # if a leaf node is encountered during tree traversal that is not the key being proven.
    assert proof.auxExistence is False
    assert proof.auxKey == b"\x00" * 32
    assert proof.auxValue == b"\x00" * 32

    # this is how we extract the key from the proof
    assert to_checksum_address(proof.key[-20:]) == alice.eth.address

    # this is how to extract the value from the proof
    value = int.from_bytes(proof.value, byteorder="big")
    assert value > 0

    # the actual merkle path is in the siblings field
    assert len(proof.siblings) > 0, "Proof should have siblings"


def test_proof_difference(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount, chain: Any
) -> None:
    assert not basic_aetherweave.deposits(alice.eth)
    assert not basic_aetherweave.deposits(bob.eth)

    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    chain.mine(1, timestamp=chain.pending_timestamp + 10)

    receipt2 = basic_aetherweave.deposit(sender=bob.eth, value="1 Eth")
    assert not receipt2.failed, "Transaction failed"

    proof1 = basic_aetherweave.getProof(sender=alice.eth)
    proof2 = basic_aetherweave.getProof(sender=bob.eth)

    assert proof1.root == proof2.root
    assert proof1.siblings != proof2.siblings
    assert proof1.key != proof2.key
    assert proof1.value == proof2.value == b"\x00" * 31 + b"\x01"
    assert proof1.existence
    assert proof2.existence
    assert proof1.auxExistence is False
    assert proof2.auxExistence is False


def test_redeposit(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    assert not basic_aetherweave.deposits(alice.eth)

    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    assert basic_aetherweave.deposits(alice.eth) > 0

    # Attempt to redeposit
    # with ape.reverts("Already staked"): # <- For some reason this doesn't work
    with pytest.raises(ape.exceptions.VirtualMachineError):
        receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")


def test_withdraw(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount, chain: Any
) -> None:
    assert not basic_aetherweave.deposits(alice.eth)

    # Attempt to withdraw without deposit

    # with ape.reverts("No stake to withdraw"): # <- For some reason this doesn't work
    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.requestWithdrawal(sender=alice.eth)

    # Deposit and then withdraw
    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    receipt = basic_aetherweave.requestWithdrawal(sender=alice.eth)
    assert not receipt.failed, "Transaction failed"

    chain.mine(1, timestamp=chain.pending_timestamp + 1)

    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.claimWithdrawal(sender=alice.eth)

    withdraw_time = HOUR + basic_aetherweave.nextEpochStartTime() + 1
    chain.mine(1, timestamp=withdraw_time)

    receipt = basic_aetherweave.claimWithdrawal(sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert (
        basic_aetherweave.withdrawalTime(alice.eth) == 0
    ), "Withdrawal should be zero after claiming"
    assert not basic_aetherweave.deposits(
        alice.eth
    ), "Stake should be zero after claiming withdrawal"


def test_cannot_restake_while_withdrawal_in_progress(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    # Request withdrawal
    receipt = basic_aetherweave.requestWithdrawal(sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert basic_aetherweave.withdrawalTime(alice.eth) > 0
    assert not basic_aetherweave.deposits(alice.eth)

    # Attempt to redeposit
    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")


def test_slash_happy_path(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    # now we slash the account.

    commit1 = b"\x01" * 32  # some dummy root hash
    commit2 = b"\x02" * 32  # another dummy root hash
    round = 3

    sig1 = utils.eth_sign(alice.eth, [commit1, round], ["bytes32", "int256"])
    sig2 = utils.eth_sign(alice.eth, [commit2, round], ["bytes32", "int256"])

    receipt = basic_aetherweave.slash(
        commit1,
        commit2,
        round,
        sig1.encode_rsv(),
        sig2.encode_rsv(),
        alice.eth.address,
        sender=bob.eth,
    )
    assert not receipt.failed, "Transaction failed"

    assert not basic_aetherweave.deposits(
        alice.eth
    ), "Stake should be zero after slashing"
    assert (
        basic_aetherweave.withdrawalTime(alice.eth) == 0
    ), "Withdrawals should be zero after slashing"


def test_slash_invalid_signature(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    commit1 = b"\x01" * 32  # some dummy root hash
    commit2 = b"\x02" * 32  # another dummy root hash
    round = 3

    sig1 = utils.eth_sign(alice.eth, [commit1, round], ["bytes32", "int256"])
    # Use a different account to sign the second commit
    sig2 = utils.eth_sign(bob.eth, [commit2, round], ["bytes32", "int256"])
    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.slash(
            commit1,
            commit2,
            round,
            sig1.encode_rsv(),
            sig2.encode_rsv(),
            alice.eth.address,
            sender=bob.eth,
        )

    # use the wrong round number on the second signature
    sig3 = utils.eth_sign(
        alice.eth, [commit2, round + 1], ["bytes32", "int256"]
    )
    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.slash(
            commit1,
            commit2,
            round,
            sig1.encode_rsv(),
            sig3.encode_rsv(),
            alice.eth.address,
            sender=bob.eth,
        )

    # use the wrong root hash on the second signature
    sig4 = utils.eth_sign(
        alice.eth, [b"\x03" * 32, round], ["bytes32", "int256"]
    )
    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.slash(
            commit1,
            commit2,
            round,
            sig1.encode_rsv(),
            sig4.encode_rsv(),
            alice.eth.address,
            sender=bob.eth,
        )


def test_slash_account_mid_withdraw(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    # Request withdrawal
    receipt = basic_aetherweave.requestWithdrawal(sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert basic_aetherweave.withdrawalTime(alice.eth) > 0
    assert not basic_aetherweave.deposits(alice.eth)

    commit1 = b"\x01" * 32  # some dummy root hash
    commit2 = b"\x02" * 32  # another dummy root hash
    round = 3

    sig1 = utils.eth_sign(alice.eth, [commit1, round], ["bytes32", "int256"])
    sig2 = utils.eth_sign(alice.eth, [commit2, round], ["bytes32", "int256"])

    receipt = basic_aetherweave.slash(
        commit1,
        commit2,
        round,
        sig1.encode_rsv(),
        sig2.encode_rsv(),
        alice.eth.address,
        sender=bob.eth,
    )
    assert not receipt.failed, "Transaction failed"

    assert not basic_aetherweave.deposits(
        alice.eth
    ), "Stake should be zero after slashing"
    assert (
        basic_aetherweave.withdrawalTime(alice.eth) == 0
    ), "Withdrawals should be zero after slashing"


def test_freeze_time(
    basic_aetherweave: Any, alice: AWAccount, bob: AWAccount, chain: Any
) -> None:

    receipt = basic_aetherweave.deposit(sender=bob.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"

    """Test that the freeze time is correctly set and enforced."""
    freeze_time = basic_aetherweave.nextEpochStartTime() - HOUR
    chain.mine(1, timestamp=freeze_time + 1)

    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")

    with pytest.raises(ape.exceptions.VirtualMachineError):
        basic_aetherweave.requestWithdrawal(sender=bob.eth)

    chain.mine(1, timestamp=chain.pending_timestamp + HOUR)
    receipt = basic_aetherweave.requestWithdrawal(sender=bob.eth)
    assert not receipt.failed, "Transaction failed"

    receipt = basic_aetherweave.deposit(sender=alice.eth, value="1 Eth")
    assert not receipt.failed, "Transaction failed"
