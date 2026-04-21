// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {SparseMerkleTree} from "./lib/SparseMerkleTree.sol";
import {PoseidonT2} from "./lib/PoseidonT2.sol";
import {PoseidonT3} from "./lib/PoseidonT3.sol";
import {PoseidonT4} from "./lib/PoseidonT4.sol";

contract AetherWeavePrivate {
    mapping(uint256 => bool) public deposits;
    mapping(uint256 => uint256) public withdrawalTime;
    mapping(uint256 => address payable) public owner;
    uint256 public immutable stakeUnit;
    uint256 public immutable epochLength;
    uint256 public immutable withdrawalDelay;
    uint256 public immutable stakeFreezePeriod;

    using SparseMerkleTree for SparseMerkleTree.UintSMT;
    SparseMerkleTree.UintSMT internal _tree;

    constructor(
        uint32 _maxTreeDepth,
        uint256 _stakeUnit,
        uint256 _epochLength,
        uint256 _withdrawalDelay,
        uint256 _stakeFreezePeriod
    ) {
        // Initialize the Sparse Merkle Tree
        _tree.initialize(_maxTreeDepth);
        _tree.setHashers(poseidon2, poseidon3);
        stakeUnit = _stakeUnit;
        epochLength = _epochLength;
        withdrawalDelay = _withdrawalDelay;
        stakeFreezePeriod = _stakeFreezePeriod;
    }

    function currentEpoch() public view returns (uint256) {
        return block.timestamp / epochLength;
    }

    function nextEpochStartTime() public view returns (uint256) {
        return (block.timestamp / epochLength + 1) * epochLength;
    }

    modifier notDuringFreezePeriod() {
        require(
            block.timestamp < nextEpochStartTime() - stakeFreezePeriod,
            "Cannot perform this action during the freeze period"
        );
        _;
    }

    modifier onlyOwner(uint256 _stakeID) {
        require(owner[_stakeID] == msg.sender, "Not the owner of the stake");
        _;
    }

    /*
        Stake and Withdrawal functions
    */
    function deposit(uint256 _stakeID) external payable notDuringFreezePeriod {
        require(!deposits[_stakeID], "This Stake Id is already used");
        require(withdrawalTime[_stakeID] == 0, "Withdrawal in progress");
        require(msg.value == stakeUnit, "Deposit must be a single stake unit");
        deposits[_stakeID] = true;
        owner[_stakeID] = payable(msg.sender);
        _tree.add(bytes32(_stakeID), 1);
    }

    function requestWithdrawal(
        uint256 _stakeID
    ) external onlyOwner(_stakeID) notDuringFreezePeriod {
        require(deposits[_stakeID], "No stake to withdraw");
        require(
            withdrawalTime[_stakeID] == 0,
            "Withdrawal already in progress"
        );
        withdrawalTime[_stakeID] = nextEpochStartTime() + withdrawalDelay;
        _tree.remove(bytes32(_stakeID));
        delete deposits[_stakeID];
    }

    function claimWithdrawal(uint256 _stakeID) external onlyOwner(_stakeID) {
        require(withdrawalTime[_stakeID] > 0, "No withdrawal to claim");
        require(
            block.timestamp >= withdrawalTime[_stakeID],
            "Withdrawal not ready"
        );
        delete withdrawalTime[_stakeID];
        delete owner[_stakeID];
        payable(msg.sender).transfer(stakeUnit);
    }

    function slash(uint256 _stakeSecret, uint256 _stakeID) external {
        require(
            poseidon1(bytes32(_stakeSecret)) == bytes32(_stakeID),
            "stakeSecret is not a preimage of stakeID"
        );
        if (deposits[_stakeID]) {
            delete deposits[_stakeID];
            _tree.remove(bytes32(_stakeID));
        } else {
            require(withdrawalTime[_stakeID] > 0, "Slashee must have a stake");
            delete withdrawalTime[_stakeID];
        }
        delete owner[_stakeID];
    }

    /*
        Public view functions
    */
    function getProof(
        uint256 _stakeID
    ) public view returns (SparseMerkleTree.Proof memory) {
        require(deposits[_stakeID], "No stake to get proof");
        return _tree.getProof(bytes32(_stakeID));
    }

    function getRoot() public view returns (bytes32) {
        return _tree.getRoot();
    }

    function verifyProof(
        SparseMerkleTree.Proof memory proof
    ) public view returns (bool) {
        return _tree.verifyProof(proof);
    }

    /*
        Helper functions
    */
    function poseidon1(bytes32 el1_) public pure returns (bytes32) {
        return bytes32(PoseidonT2.hash([uint256(el1_)]));
    }

    function poseidon2(
        bytes32 el1_,
        bytes32 el2_
    ) public pure returns (bytes32) {
        return bytes32(PoseidonT3.hash([uint256(el1_), uint256(el2_)]));
    }

    function poseidon3(
        bytes32 el1_,
        bytes32 el2_,
        bytes32 el3_
    ) public pure returns (bytes32) {
        return
            bytes32(
                PoseidonT4.hash([uint256(el1_), uint256(el2_), uint256(el3_)])
            );
    }
}
