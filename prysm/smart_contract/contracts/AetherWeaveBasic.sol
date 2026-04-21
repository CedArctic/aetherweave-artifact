// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {SparseMerkleTree} from "./lib/SparseMerkleTree.sol";
import {PoseidonT3} from "./lib/PoseidonT3.sol";
import {PoseidonT4} from "./lib/PoseidonT4.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

contract AetherWeaveBasic {
    mapping(address => bool) public deposits;
    mapping(address => uint256) public withdrawalTime;
    uint256 public immutable stakeUnit;
    uint256 public immutable epochLength;
    uint256 public immutable withdrawalDelay;
    uint256 public immutable stakeFreezePeriod;

    using ECDSA for bytes32;
    using MessageHashUtils for bytes32;

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

    /*
        Stake and Withdrawal functions
    */
    function deposit() external payable notDuringFreezePeriod {
        require(!deposits[msg.sender], "Already staked");
        require(withdrawalTime[msg.sender] == 0, "Withdrawal in progress");
        require(msg.value == stakeUnit, "Deposits must be a single stake unit");
        deposits[msg.sender] = true;
        _tree.add(bytes32(uint256(uint160(msg.sender))), 1);
    }

    function requestWithdrawal() external notDuringFreezePeriod {
        require(deposits[msg.sender], "No stake to withdraw");
        require(
            withdrawalTime[msg.sender] == 0,
            "Withdrawal already in progress"
        );
        withdrawalTime[msg.sender] = nextEpochStartTime() + withdrawalDelay;
        _tree.remove(bytes32(uint256(uint160(msg.sender))));
        delete deposits[msg.sender];
    }

    function claimWithdrawal() external {
        require(withdrawalTime[msg.sender] > 0, "No withdrawal to claim");
        require(
            block.timestamp >= withdrawalTime[msg.sender],
            "Withdrawal not ready"
        );
        delete withdrawalTime[msg.sender];
        payable(msg.sender).transfer(stakeUnit);
    }

    function verifySignature(
        bytes32 rootHash,
        uint256 round,
        bytes calldata signature,
        address slashee
    ) public pure returns (bool) {
        bytes32 hash = keccak256(abi.encode(rootHash, round));
        bytes32 ethHash = hash.toEthSignedMessageHash();
        address recovered = ethHash.recover(signature);
        return recovered == slashee;
    }

    function slash(
        bytes32 root1,
        bytes32 root2,
        uint256 round,
        bytes calldata signature1,
        bytes calldata signature2,
        address slashee
    ) external {
        require(root1 != root2, "Root hashes must be different");
        require(
            verifySignature(root1, round, signature1, slashee),
            "Invalid signature for commitment 1"
        );
        require(
            verifySignature(root2, round, signature2, slashee),
            "Invalid signature for commitment 2"
        );
        if (deposits[slashee]) {
            delete deposits[slashee];
            _tree.remove(bytes32(uint256(uint160(slashee))));
        } else {
            require(withdrawalTime[slashee] > 0, "Slashee must have a stake");
            delete withdrawalTime[slashee];
        }
    }

    /*
        Public view functions
    */
    function getProof() public view returns (SparseMerkleTree.Proof memory) {
        require(deposits[msg.sender], "No stake to get proof");
        return _tree.getProof(bytes32(uint256(uint160(msg.sender))));
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
