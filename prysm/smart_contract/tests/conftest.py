from typing import Any

import pytest
from utils import DAY, HOUR, AWAccount, babyjub_public_key, poseidon


@pytest.fixture(scope="session")
def owner(accounts: Any) -> Any:
    return accounts[0]


@pytest.fixture(scope="session")
def basic_aetherweave(
    owner: Any, project: Any, compilers: Any, chain: Any
) -> Any:
    libPoseidon2 = owner.deploy(project.PoseidonT3)
    libPoseidon3 = owner.deploy(project.PoseidonT4)
    compilers.solidity.add_library(libPoseidon2)
    compilers.solidity.add_library(libPoseidon3)

    # Parameters for AetherWeavePrivate
    max_tree_depth = 16
    stake_unit = "1 Eth"
    _epoch_length = DAY
    _withdrawal_delay = HOUR
    _stake_freeeze_period = HOUR

    basic_aetherweave = owner.deploy(
        project.AetherWeaveBasic,
        max_tree_depth,
        stake_unit,
        _epoch_length,
        _withdrawal_delay,
        _stake_freeeze_period,
    )
    epoch_start = basic_aetherweave.nextEpochStartTime()
    chain.pending_timestamp = epoch_start + 1
    return basic_aetherweave


@pytest.fixture(scope="session")
def aetherweave(owner: Any, project: Any, compilers: Any, chain: Any) -> Any:
    libPoseidon1 = owner.deploy(project.PoseidonT2)
    libPoseidon2 = owner.deploy(project.PoseidonT3)
    libPoseidon3 = owner.deploy(project.PoseidonT4)
    compilers.solidity.add_library(libPoseidon1)
    compilers.solidity.add_library(libPoseidon2)
    compilers.solidity.add_library(libPoseidon3)

    max_tree_depth = 16
    stake_unit = "1 Eth"
    _epoch_length = DAY
    _withdrawal_delay = HOUR
    _stake_freeeze_period = HOUR
    aetherweave = owner.deploy(
        project.AetherWeavePrivate,
        max_tree_depth,
        stake_unit,
        _epoch_length,
        _withdrawal_delay,
        _stake_freeeze_period,
    )
    epoch_start = aetherweave.nextEpochStartTime()
    chain.pending_timestamp = epoch_start + 1
    return aetherweave


@pytest.fixture(scope="session")
def baccounts(accounts: Any) -> list[AWAccount]:
    # using fixed secrets for testing rather than random to ensure repeatability
    secrets = [12, 34, 56, 78, 90, 123, 456, 789, 101112, 131415]
    keys = []
    for i in range(10):
        sk = secrets[i]
        account = {
            "secret": sk,
            "stakeSecret": poseidon(sk),
            "stakeID": poseidon(poseidon(sk)),
            "netPK": list(babyjub_public_key(sk)),
        }
        keys.append(account)

    baccounts = [
        AWAccount(
            eth=accounts[i],
            stakeID=poseidon(poseidon(secrets[i])),
            secret=secrets[i],
            stakeSecret=poseidon(secrets[i]),
            netPK=babyjub_public_key(secrets[i]),
        )
        for i in range(10)
    ]

    return baccounts


@pytest.fixture(scope="session")
def alice(baccounts: list[AWAccount]) -> AWAccount:
    return baccounts[1]


@pytest.fixture(scope="session")
def bob(baccounts: list[AWAccount]) -> AWAccount:
    return baccounts[2]


@pytest.fixture(scope="session")
def charlie(baccounts: list[AWAccount]) -> AWAccount:
    return baccounts[3]
