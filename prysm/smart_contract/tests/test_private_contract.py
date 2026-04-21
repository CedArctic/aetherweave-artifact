from typing import Any

import ape
import pytest
from utils import (
    DAY,
    HOUR,
    ZERO_ADDRESS,
    AWAccount,
    babyjub_public_key,
    poseidon,
)


def test_key_gen() -> None:
    secret = 12311475832582348398
    stake_secret = poseidon(secret)
    stake_id = poseidon(poseidon(secret))
    net_pk = babyjub_public_key(secret)
    assert (
        net_pk[0]
        == 11996383291524416042564348934214284640935568107003179663024246428690776956964
    )
    assert (
        net_pk[1]
        == 2576294389378298507761114155837686526018660307353259114148349782074442655070
    )
    assert (
        stake_secret
        == 3509048756036692992817569849754298161463747629421038493366100554291751420515
    )
    assert (
        stake_id
        == 8485939556704454397298135349967189213459372662393172226978955926993892542480
    )

    secret2 = 5
    stake_secret2 = poseidon(secret2)
    stake_id2 = poseidon(poseidon(secret2))
    net_pk2 = babyjub_public_key(secret2)
    assert (
        net_pk2[0]
        == 11480966271046430430613841218147196773252373073876138147006741179837832100836
    )
    assert (
        net_pk2[1]
        == 15148236048131954717802795400425086368006776860859772698778589175317365693546
    )
    assert (
        stake_secret2
        == 19065150524771031435284970883882288895168425523179566388456001105768498065277
    )
    assert (
        stake_id2
        == 19431582593299833793509148692859968279834255791370208283748698028603610032058
    )


def test_deposit_withdraw_happy_path(
    aetherweave: Any, alice: AWAccount
) -> None:
    assert not aetherweave.deposits(alice.stakeID)
    assert aetherweave.withdrawalTime(alice.stakeID) == 0

    with ape.reverts("No stake to get proof"):
        aetherweave.getProof(alice.stakeID, sender=alice.eth)

    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    assert aetherweave.owner(alice.stakeID) == alice.eth.address
    assert aetherweave.deposits(alice.stakeID)
    assert aetherweave.withdrawalTime(alice.stakeID) == 0

    proof = aetherweave.getProof(alice.stakeID, sender=alice.eth)
    assert aetherweave.verifyProof(proof, sender=alice.eth) is True
    proof.root = bytes([15] * 32)  # tamper with the proof
    assert aetherweave.verifyProof(proof, sender=alice.eth) is False

    receipt = aetherweave.requestWithdrawal(alice.stakeID, sender=alice.eth)
    assert not receipt.failed, "Transaction failed"

    assert aetherweave.withdrawalTime(alice.stakeID) > 0


def test_proof(
    aetherweave: Any, alice: AWAccount, bob: AWAccount, charlie: AWAccount
) -> None:
    receipt1 = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    receipt2 = aetherweave.deposit(bob.stakeID, sender=bob.eth, value="1 Eth")
    receipt3 = aetherweave.deposit(
        charlie.stakeID, sender=charlie.eth, value="1 Eth"
    )

    assert not receipt1.failed
    assert not receipt2.failed
    assert not receipt3.failed

    proof = aetherweave.getProof(alice.stakeID, sender=alice.eth)

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

    root = aetherweave.getRoot()
    assert proof.root == root

    assert proof.existence is True

    # the Auxiliary fields should be empty for an inclusion proof. They are only used for exclusion proofs,
    # if a leaf node is encountered during tree traversal that is not the key being proven.
    assert proof.auxExistence is False
    assert proof.auxKey == b"\x00" * 32
    assert proof.auxValue == b"\x00" * 32

    # this is how we extract the key from the proof
    assert int.from_bytes(proof.key, byteorder="big") == alice.stakeID

    # this is how to extract the value from the proof
    value = int.from_bytes(proof.value, byteorder="big")
    assert value > 0

    # the actual merkle path is in the siblings field
    assert len(proof.siblings) > 0, "Proof should have siblings"


def test_proof_difference(
    aetherweave: Any, chain: Any, alice: AWAccount, bob: AWAccount
) -> None:
    assert not aetherweave.deposits(alice.stakeID)
    assert not aetherweave.deposits(bob.stakeID)
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    chain.mine(1, timestamp=chain.pending_timestamp + 10)

    receipt2 = aetherweave.deposit(bob.stakeID, sender=bob.eth, value="1 Eth")
    assert not receipt2.failed, "Transaction failed"

    proof1 = aetherweave.getProof(alice.stakeID, sender=alice.eth)
    proof2 = aetherweave.getProof(bob.stakeID, sender=bob.eth)

    assert proof1.root == proof2.root
    assert proof1.siblings != proof2.siblings
    assert proof1.key != proof2.key
    assert proof1.value == proof2.value == b"\x00" * 31 + b"\x01"
    assert proof1.existence
    assert proof2.existence
    assert proof1.auxExistence is False
    assert proof2.auxExistence is False


def test_redeposit(aetherweave: Any, alice: AWAccount, bob: AWAccount) -> None:
    assert not aetherweave.deposits(alice.stakeID)

    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    assert aetherweave.deposits(alice.stakeID)

    # Attempt to redeposit from two different senders
    with pytest.raises(ape.exceptions.VirtualMachineError):
        receipt = aetherweave.deposit(
            alice.stakeID, sender=alice.eth, value="1 Eth"
        )
    with pytest.raises(ape.exceptions.VirtualMachineError):
        receipt = aetherweave.deposit(
            alice.stakeID, sender=bob.eth, value="1 Eth"
        )


def test_withdraw(
    aetherweave: Any, alice: AWAccount, bob: AWAccount, chain: Any
) -> None:
    assert not aetherweave.deposits(alice.stakeID)

    # Attempt to withdraw without deposit

    # with ape.reverts("No stake to withdraw"): # <- For some reason this doesn't work
    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.requestWithdrawal(alice.stakeID, sender=alice.eth)

    # Deposit and then withdraw
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    receipt = aetherweave.requestWithdrawal(alice.stakeID, sender=alice.eth)
    assert not receipt.failed, "Transaction failed"

    chain.mine(1, timestamp=chain.pending_timestamp + 3600)

    # Can't claim someone else's withdrawal
    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.claimWithdrawal(alice.stakeID, sender=bob.eth)

    # with ape.reverts("No stake to withdraw"):  # <- For some reason this doesn't work
    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.claimWithdrawal(alice.stakeID, sender=alice.eth)

    chain.mine(1, timestamp=chain.pending_timestamp + DAY)

    receipt = aetherweave.claimWithdrawal(alice.stakeID, sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert (
        aetherweave.withdrawalTime(alice.stakeID) == 0
    ), "Withdrawal should be zero after claiming"
    assert not aetherweave.deposits(
        alice.stakeID
    ), "Stake should be zero after claiming withdrawal"
    assert aetherweave.owner(alice.stakeID) == ZERO_ADDRESS


def test_cannot_restake_while_withdrawal_in_progress(
    aetherweave: Any, alice: AWAccount
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    # Request withdrawal
    receipt = aetherweave.requestWithdrawal(alice.stakeID, sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert aetherweave.withdrawalTime(alice.stakeID) > 0
    assert not aetherweave.deposits(alice.stakeID)

    # Attempt to redeposit
    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.deposit(alice.stakeID, sender=alice.eth, value="1 Eth")


def test_slash_happy_path(
    aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    # now we slash the account.
    receipt = aetherweave.slash(
        alice.stakeSecret,
        alice.stakeID,
        sender=bob.eth,
    )
    assert not receipt.failed, "Transaction failed"

    assert not aetherweave.deposits(
        alice.stakeID
    ), "Stake should be zero after slashing"
    assert (
        aetherweave.withdrawalTime(alice.stakeID) == 0
    ), "Withdrawals should be zero after slashing"


def test_slash_invalid_preimage(
    aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.slash(
            alice.stakeSecret,
            77545454,
            sender=bob.eth,
        )

    with pytest.raises(ape.exceptions.VirtualMachineError):
        aetherweave.slash(
            12345678901234567890,
            alice.stakeID,
            sender=bob.eth,
        )


def test_slash_account_mid_withdraw(
    aetherweave: Any, alice: AWAccount, bob: AWAccount
) -> None:
    receipt = aetherweave.deposit(
        alice.stakeID, sender=alice.eth, value="1 Eth"
    )
    assert not receipt.failed, "Transaction failed"

    # Request withdrawal
    receipt = aetherweave.requestWithdrawal(alice.stakeID, sender=alice.eth)
    assert not receipt.failed, "Transaction failed"
    assert aetherweave.withdrawalTime(alice.stakeID) > 0
    assert not aetherweave.deposits(alice.stakeID)

    receipt = aetherweave.slash(
        alice.stakeSecret,
        alice.stakeID,
        sender=bob.eth,
    )
    assert not receipt.failed, "Transaction failed"

    assert not aetherweave.deposits(
        alice.stakeID
    ), "Stake should be zero after slashing"
    assert (
        aetherweave.withdrawalTime(alice.stakeID) == 0
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
