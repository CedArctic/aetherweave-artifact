// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package awcontract

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// SparseMerkleTreeProof is an auto generated low-level Go binding around an user-defined struct.
type SparseMerkleTreeProof struct {
	Root         [32]byte
	Siblings     [][32]byte
	Existence    bool
	Key          [32]byte
	Value        [32]byte
	AuxExistence bool
	AuxKey       [32]byte
	AuxValue     [32]byte
}

// AwcontractMetaData contains all meta data concerning the Awcontract contract.
var AwcontractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_maxTreeDepth\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"_stakeUnit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_epochLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_withdrawalDelay\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_stakeFreezePeriod\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"}],\"name\":\"KeyAlreadyExists\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"currentKey\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"}],\"name\":\"LeafDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"maxDepth\",\"type\":\"uint32\"}],\"name\":\"MaxDepthExceedsHardCap\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxDepthIsZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxDepthReached\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"currentDepth\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"newDepth\",\"type\":\"uint32\"}],\"name\":\"NewMaxDepthMustBeLarger\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nodeId\",\"type\":\"uint256\"}],\"name\":\"NodeDoesNotExist\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TreeAlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TreeIsNotEmpty\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TreeNotInitialized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"}],\"name\":\"DebugMsg\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"v1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"v2\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"a\",\"type\":\"address\"}],\"name\":\"DebugPoint\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"claimWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"debug_tree_depth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"debug_tree_root\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"deposit_debug\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"epochLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"getProof\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"bool\",\"name\":\"existence\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"auxExistence\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"auxKey\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"auxValue\",\"type\":\"bytes32\"}],\"internalType\":\"structSparseMerkleTree.Proof\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_currentEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_epochLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_nextEpochStartTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_stakeFreezePeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_stakeUnit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_withdrawalDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_maxTreeDepth\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"_stakeUnit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_epochLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_withdrawalDelay\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_stakeFreezePeriod\",\"type\":\"uint256\"}],\"name\":\"init\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextEpochStartTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"addresspayable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"el1_\",\"type\":\"bytes32\"}],\"name\":\"poseidon1\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"el1_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"el2_\",\"type\":\"bytes32\"}],\"name\":\"poseidon2\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"el1_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"el2_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"el3_\",\"type\":\"bytes32\"}],\"name\":\"poseidon3\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"requestWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_stakeSecret\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_stakeID\",\"type\":\"uint256\"}],\"name\":\"slash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeFreezePeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeUnit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"bool\",\"name\":\"existence\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"auxExistence\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"auxKey\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"auxValue\",\"type\":\"bytes32\"}],\"internalType\":\"structSparseMerkleTree.Proof\",\"name\":\"proof\",\"type\":\"tuple\"}],\"name\":\"verifyProof\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawalDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"withdrawalTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AwcontractABI is the input ABI used to generate the binding from.
// Deprecated: Use AwcontractMetaData.ABI instead.
var AwcontractABI = AwcontractMetaData.ABI

// Awcontract is an auto generated Go binding around an Ethereum contract.
type Awcontract struct {
	AwcontractCaller     // Read-only binding to the contract
	AwcontractTransactor // Write-only binding to the contract
	AwcontractFilterer   // Log filterer for contract events
}

// AwcontractCaller is an auto generated read-only Go binding around an Ethereum contract.
type AwcontractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AwcontractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AwcontractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AwcontractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AwcontractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AwcontractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AwcontractSession struct {
	Contract     *Awcontract       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AwcontractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AwcontractCallerSession struct {
	Contract *AwcontractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AwcontractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AwcontractTransactorSession struct {
	Contract     *AwcontractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AwcontractRaw is an auto generated low-level Go binding around an Ethereum contract.
type AwcontractRaw struct {
	Contract *Awcontract // Generic contract binding to access the raw methods on
}

// AwcontractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AwcontractCallerRaw struct {
	Contract *AwcontractCaller // Generic read-only contract binding to access the raw methods on
}

// AwcontractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AwcontractTransactorRaw struct {
	Contract *AwcontractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAwcontract creates a new instance of Awcontract, bound to a specific deployed contract.
func NewAwcontract(address common.Address, backend bind.ContractBackend) (*Awcontract, error) {
	contract, err := bindAwcontract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Awcontract{AwcontractCaller: AwcontractCaller{contract: contract}, AwcontractTransactor: AwcontractTransactor{contract: contract}, AwcontractFilterer: AwcontractFilterer{contract: contract}}, nil
}

// NewAwcontractCaller creates a new read-only instance of Awcontract, bound to a specific deployed contract.
func NewAwcontractCaller(address common.Address, caller bind.ContractCaller) (*AwcontractCaller, error) {
	contract, err := bindAwcontract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AwcontractCaller{contract: contract}, nil
}

// NewAwcontractTransactor creates a new write-only instance of Awcontract, bound to a specific deployed contract.
func NewAwcontractTransactor(address common.Address, transactor bind.ContractTransactor) (*AwcontractTransactor, error) {
	contract, err := bindAwcontract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AwcontractTransactor{contract: contract}, nil
}

// NewAwcontractFilterer creates a new log filterer instance of Awcontract, bound to a specific deployed contract.
func NewAwcontractFilterer(address common.Address, filterer bind.ContractFilterer) (*AwcontractFilterer, error) {
	contract, err := bindAwcontract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AwcontractFilterer{contract: contract}, nil
}

// bindAwcontract binds a generic wrapper to an already deployed contract.
func bindAwcontract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AwcontractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Awcontract *AwcontractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Awcontract.Contract.AwcontractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Awcontract *AwcontractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Awcontract.Contract.AwcontractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Awcontract *AwcontractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Awcontract.Contract.AwcontractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Awcontract *AwcontractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Awcontract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Awcontract *AwcontractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Awcontract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Awcontract *AwcontractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Awcontract.Contract.contract.Transact(opts, method, params...)
}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractCaller) CurrentEpoch(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "currentEpoch")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractSession) CurrentEpoch() (*big.Int, error) {
	return _Awcontract.Contract.CurrentEpoch(&_Awcontract.CallOpts)
}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) CurrentEpoch() (*big.Int, error) {
	return _Awcontract.Contract.CurrentEpoch(&_Awcontract.CallOpts)
}

// DebugTreeDepth is a free data retrieval call binding the contract method 0x7a57a142.
//
// Solidity: function debug_tree_depth() view returns(uint256)
func (_Awcontract *AwcontractCaller) DebugTreeDepth(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "debug_tree_depth")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DebugTreeDepth is a free data retrieval call binding the contract method 0x7a57a142.
//
// Solidity: function debug_tree_depth() view returns(uint256)
func (_Awcontract *AwcontractSession) DebugTreeDepth() (*big.Int, error) {
	return _Awcontract.Contract.DebugTreeDepth(&_Awcontract.CallOpts)
}

// DebugTreeDepth is a free data retrieval call binding the contract method 0x7a57a142.
//
// Solidity: function debug_tree_depth() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) DebugTreeDepth() (*big.Int, error) {
	return _Awcontract.Contract.DebugTreeDepth(&_Awcontract.CallOpts)
}

// DebugTreeRoot is a free data retrieval call binding the contract method 0x50ce6278.
//
// Solidity: function debug_tree_root() view returns(bytes32)
func (_Awcontract *AwcontractCaller) DebugTreeRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "debug_tree_root")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DebugTreeRoot is a free data retrieval call binding the contract method 0x50ce6278.
//
// Solidity: function debug_tree_root() view returns(bytes32)
func (_Awcontract *AwcontractSession) DebugTreeRoot() ([32]byte, error) {
	return _Awcontract.Contract.DebugTreeRoot(&_Awcontract.CallOpts)
}

// DebugTreeRoot is a free data retrieval call binding the contract method 0x50ce6278.
//
// Solidity: function debug_tree_root() view returns(bytes32)
func (_Awcontract *AwcontractCallerSession) DebugTreeRoot() ([32]byte, error) {
	return _Awcontract.Contract.DebugTreeRoot(&_Awcontract.CallOpts)
}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(bool)
func (_Awcontract *AwcontractCaller) Deposits(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "deposits", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(bool)
func (_Awcontract *AwcontractSession) Deposits(arg0 *big.Int) (bool, error) {
	return _Awcontract.Contract.Deposits(&_Awcontract.CallOpts, arg0)
}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(bool)
func (_Awcontract *AwcontractCallerSession) Deposits(arg0 *big.Int) (bool, error) {
	return _Awcontract.Contract.Deposits(&_Awcontract.CallOpts, arg0)
}

// EpochLength is a free data retrieval call binding the contract method 0x57d775f8.
//
// Solidity: function epochLength() view returns(uint256)
func (_Awcontract *AwcontractCaller) EpochLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "epochLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EpochLength is a free data retrieval call binding the contract method 0x57d775f8.
//
// Solidity: function epochLength() view returns(uint256)
func (_Awcontract *AwcontractSession) EpochLength() (*big.Int, error) {
	return _Awcontract.Contract.EpochLength(&_Awcontract.CallOpts)
}

// EpochLength is a free data retrieval call binding the contract method 0x57d775f8.
//
// Solidity: function epochLength() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) EpochLength() (*big.Int, error) {
	return _Awcontract.Contract.EpochLength(&_Awcontract.CallOpts)
}

// GetProof is a free data retrieval call binding the contract method 0x11149ada.
//
// Solidity: function getProof(uint256 _stakeID) view returns((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32))
func (_Awcontract *AwcontractCaller) GetProof(opts *bind.CallOpts, _stakeID *big.Int) (SparseMerkleTreeProof, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "getProof", _stakeID)

	if err != nil {
		return *new(SparseMerkleTreeProof), err
	}

	out0 := *abi.ConvertType(out[0], new(SparseMerkleTreeProof)).(*SparseMerkleTreeProof)

	return out0, err

}

// GetProof is a free data retrieval call binding the contract method 0x11149ada.
//
// Solidity: function getProof(uint256 _stakeID) view returns((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32))
func (_Awcontract *AwcontractSession) GetProof(_stakeID *big.Int) (SparseMerkleTreeProof, error) {
	return _Awcontract.Contract.GetProof(&_Awcontract.CallOpts, _stakeID)
}

// GetProof is a free data retrieval call binding the contract method 0x11149ada.
//
// Solidity: function getProof(uint256 _stakeID) view returns((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32))
func (_Awcontract *AwcontractCallerSession) GetProof(_stakeID *big.Int) (SparseMerkleTreeProof, error) {
	return _Awcontract.Contract.GetProof(&_Awcontract.CallOpts, _stakeID)
}

// GetRoot is a free data retrieval call binding the contract method 0x5ca1e165.
//
// Solidity: function getRoot() view returns(bytes32)
func (_Awcontract *AwcontractCaller) GetRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "getRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoot is a free data retrieval call binding the contract method 0x5ca1e165.
//
// Solidity: function getRoot() view returns(bytes32)
func (_Awcontract *AwcontractSession) GetRoot() ([32]byte, error) {
	return _Awcontract.Contract.GetRoot(&_Awcontract.CallOpts)
}

// GetRoot is a free data retrieval call binding the contract method 0x5ca1e165.
//
// Solidity: function getRoot() view returns(bytes32)
func (_Awcontract *AwcontractCallerSession) GetRoot() ([32]byte, error) {
	return _Awcontract.Contract.GetRoot(&_Awcontract.CallOpts)
}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0xe7ecd11f.
//
// Solidity: function get_currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetCurrentEpoch(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_currentEpoch")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0xe7ecd11f.
//
// Solidity: function get_currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractSession) GetCurrentEpoch() (*big.Int, error) {
	return _Awcontract.Contract.GetCurrentEpoch(&_Awcontract.CallOpts)
}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0xe7ecd11f.
//
// Solidity: function get_currentEpoch() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetCurrentEpoch() (*big.Int, error) {
	return _Awcontract.Contract.GetCurrentEpoch(&_Awcontract.CallOpts)
}

// GetEpochLength is a free data retrieval call binding the contract method 0xfb8d1ea1.
//
// Solidity: function get_epochLength() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetEpochLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_epochLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochLength is a free data retrieval call binding the contract method 0xfb8d1ea1.
//
// Solidity: function get_epochLength() view returns(uint256)
func (_Awcontract *AwcontractSession) GetEpochLength() (*big.Int, error) {
	return _Awcontract.Contract.GetEpochLength(&_Awcontract.CallOpts)
}

// GetEpochLength is a free data retrieval call binding the contract method 0xfb8d1ea1.
//
// Solidity: function get_epochLength() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetEpochLength() (*big.Int, error) {
	return _Awcontract.Contract.GetEpochLength(&_Awcontract.CallOpts)
}

// GetNextEpochStartTime is a free data retrieval call binding the contract method 0x9c51660f.
//
// Solidity: function get_nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetNextEpochStartTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_nextEpochStartTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextEpochStartTime is a free data retrieval call binding the contract method 0x9c51660f.
//
// Solidity: function get_nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractSession) GetNextEpochStartTime() (*big.Int, error) {
	return _Awcontract.Contract.GetNextEpochStartTime(&_Awcontract.CallOpts)
}

// GetNextEpochStartTime is a free data retrieval call binding the contract method 0x9c51660f.
//
// Solidity: function get_nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetNextEpochStartTime() (*big.Int, error) {
	return _Awcontract.Contract.GetNextEpochStartTime(&_Awcontract.CallOpts)
}

// GetStakeFreezePeriod is a free data retrieval call binding the contract method 0xa9aa214b.
//
// Solidity: function get_stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetStakeFreezePeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_stakeFreezePeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStakeFreezePeriod is a free data retrieval call binding the contract method 0xa9aa214b.
//
// Solidity: function get_stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractSession) GetStakeFreezePeriod() (*big.Int, error) {
	return _Awcontract.Contract.GetStakeFreezePeriod(&_Awcontract.CallOpts)
}

// GetStakeFreezePeriod is a free data retrieval call binding the contract method 0xa9aa214b.
//
// Solidity: function get_stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetStakeFreezePeriod() (*big.Int, error) {
	return _Awcontract.Contract.GetStakeFreezePeriod(&_Awcontract.CallOpts)
}

// GetStakeUnit is a free data retrieval call binding the contract method 0x45526d14.
//
// Solidity: function get_stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetStakeUnit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_stakeUnit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStakeUnit is a free data retrieval call binding the contract method 0x45526d14.
//
// Solidity: function get_stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractSession) GetStakeUnit() (*big.Int, error) {
	return _Awcontract.Contract.GetStakeUnit(&_Awcontract.CallOpts)
}

// GetStakeUnit is a free data retrieval call binding the contract method 0x45526d14.
//
// Solidity: function get_stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetStakeUnit() (*big.Int, error) {
	return _Awcontract.Contract.GetStakeUnit(&_Awcontract.CallOpts)
}

// GetWithdrawalDelay is a free data retrieval call binding the contract method 0xa7240648.
//
// Solidity: function get_withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractCaller) GetWithdrawalDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "get_withdrawalDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawalDelay is a free data retrieval call binding the contract method 0xa7240648.
//
// Solidity: function get_withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractSession) GetWithdrawalDelay() (*big.Int, error) {
	return _Awcontract.Contract.GetWithdrawalDelay(&_Awcontract.CallOpts)
}

// GetWithdrawalDelay is a free data retrieval call binding the contract method 0xa7240648.
//
// Solidity: function get_withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) GetWithdrawalDelay() (*big.Int, error) {
	return _Awcontract.Contract.GetWithdrawalDelay(&_Awcontract.CallOpts)
}

// NextEpochStartTime is a free data retrieval call binding the contract method 0x06ea4592.
//
// Solidity: function nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractCaller) NextEpochStartTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "nextEpochStartTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextEpochStartTime is a free data retrieval call binding the contract method 0x06ea4592.
//
// Solidity: function nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractSession) NextEpochStartTime() (*big.Int, error) {
	return _Awcontract.Contract.NextEpochStartTime(&_Awcontract.CallOpts)
}

// NextEpochStartTime is a free data retrieval call binding the contract method 0x06ea4592.
//
// Solidity: function nextEpochStartTime() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) NextEpochStartTime() (*big.Int, error) {
	return _Awcontract.Contract.NextEpochStartTime(&_Awcontract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0xa123c33e.
//
// Solidity: function owner(uint256 ) view returns(address)
func (_Awcontract *AwcontractCaller) Owner(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "owner", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0xa123c33e.
//
// Solidity: function owner(uint256 ) view returns(address)
func (_Awcontract *AwcontractSession) Owner(arg0 *big.Int) (common.Address, error) {
	return _Awcontract.Contract.Owner(&_Awcontract.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0xa123c33e.
//
// Solidity: function owner(uint256 ) view returns(address)
func (_Awcontract *AwcontractCallerSession) Owner(arg0 *big.Int) (common.Address, error) {
	return _Awcontract.Contract.Owner(&_Awcontract.CallOpts, arg0)
}

// Poseidon1 is a free data retrieval call binding the contract method 0x80d0dd05.
//
// Solidity: function poseidon1(bytes32 el1_) pure returns(bytes32)
func (_Awcontract *AwcontractCaller) Poseidon1(opts *bind.CallOpts, el1_ [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "poseidon1", el1_)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Poseidon1 is a free data retrieval call binding the contract method 0x80d0dd05.
//
// Solidity: function poseidon1(bytes32 el1_) pure returns(bytes32)
func (_Awcontract *AwcontractSession) Poseidon1(el1_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon1(&_Awcontract.CallOpts, el1_)
}

// Poseidon1 is a free data retrieval call binding the contract method 0x80d0dd05.
//
// Solidity: function poseidon1(bytes32 el1_) pure returns(bytes32)
func (_Awcontract *AwcontractCallerSession) Poseidon1(el1_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon1(&_Awcontract.CallOpts, el1_)
}

// Poseidon2 is a free data retrieval call binding the contract method 0xa846a519.
//
// Solidity: function poseidon2(bytes32 el1_, bytes32 el2_) pure returns(bytes32)
func (_Awcontract *AwcontractCaller) Poseidon2(opts *bind.CallOpts, el1_ [32]byte, el2_ [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "poseidon2", el1_, el2_)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Poseidon2 is a free data retrieval call binding the contract method 0xa846a519.
//
// Solidity: function poseidon2(bytes32 el1_, bytes32 el2_) pure returns(bytes32)
func (_Awcontract *AwcontractSession) Poseidon2(el1_ [32]byte, el2_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon2(&_Awcontract.CallOpts, el1_, el2_)
}

// Poseidon2 is a free data retrieval call binding the contract method 0xa846a519.
//
// Solidity: function poseidon2(bytes32 el1_, bytes32 el2_) pure returns(bytes32)
func (_Awcontract *AwcontractCallerSession) Poseidon2(el1_ [32]byte, el2_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon2(&_Awcontract.CallOpts, el1_, el2_)
}

// Poseidon3 is a free data retrieval call binding the contract method 0xff88cbfa.
//
// Solidity: function poseidon3(bytes32 el1_, bytes32 el2_, bytes32 el3_) pure returns(bytes32)
func (_Awcontract *AwcontractCaller) Poseidon3(opts *bind.CallOpts, el1_ [32]byte, el2_ [32]byte, el3_ [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "poseidon3", el1_, el2_, el3_)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Poseidon3 is a free data retrieval call binding the contract method 0xff88cbfa.
//
// Solidity: function poseidon3(bytes32 el1_, bytes32 el2_, bytes32 el3_) pure returns(bytes32)
func (_Awcontract *AwcontractSession) Poseidon3(el1_ [32]byte, el2_ [32]byte, el3_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon3(&_Awcontract.CallOpts, el1_, el2_, el3_)
}

// Poseidon3 is a free data retrieval call binding the contract method 0xff88cbfa.
//
// Solidity: function poseidon3(bytes32 el1_, bytes32 el2_, bytes32 el3_) pure returns(bytes32)
func (_Awcontract *AwcontractCallerSession) Poseidon3(el1_ [32]byte, el2_ [32]byte, el3_ [32]byte) ([32]byte, error) {
	return _Awcontract.Contract.Poseidon3(&_Awcontract.CallOpts, el1_, el2_, el3_)
}

// StakeFreezePeriod is a free data retrieval call binding the contract method 0x16f6d547.
//
// Solidity: function stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractCaller) StakeFreezePeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "stakeFreezePeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakeFreezePeriod is a free data retrieval call binding the contract method 0x16f6d547.
//
// Solidity: function stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractSession) StakeFreezePeriod() (*big.Int, error) {
	return _Awcontract.Contract.StakeFreezePeriod(&_Awcontract.CallOpts)
}

// StakeFreezePeriod is a free data retrieval call binding the contract method 0x16f6d547.
//
// Solidity: function stakeFreezePeriod() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) StakeFreezePeriod() (*big.Int, error) {
	return _Awcontract.Contract.StakeFreezePeriod(&_Awcontract.CallOpts)
}

// StakeUnit is a free data retrieval call binding the contract method 0x8070eded.
//
// Solidity: function stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractCaller) StakeUnit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "stakeUnit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakeUnit is a free data retrieval call binding the contract method 0x8070eded.
//
// Solidity: function stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractSession) StakeUnit() (*big.Int, error) {
	return _Awcontract.Contract.StakeUnit(&_Awcontract.CallOpts)
}

// StakeUnit is a free data retrieval call binding the contract method 0x8070eded.
//
// Solidity: function stakeUnit() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) StakeUnit() (*big.Int, error) {
	return _Awcontract.Contract.StakeUnit(&_Awcontract.CallOpts)
}

// VerifyProof is a free data retrieval call binding the contract method 0xc7f547c1.
//
// Solidity: function verifyProof((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32) proof) view returns(bool)
func (_Awcontract *AwcontractCaller) VerifyProof(opts *bind.CallOpts, proof SparseMerkleTreeProof) (bool, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "verifyProof", proof)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifyProof is a free data retrieval call binding the contract method 0xc7f547c1.
//
// Solidity: function verifyProof((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32) proof) view returns(bool)
func (_Awcontract *AwcontractSession) VerifyProof(proof SparseMerkleTreeProof) (bool, error) {
	return _Awcontract.Contract.VerifyProof(&_Awcontract.CallOpts, proof)
}

// VerifyProof is a free data retrieval call binding the contract method 0xc7f547c1.
//
// Solidity: function verifyProof((bytes32,bytes32[],bool,bytes32,bytes32,bool,bytes32,bytes32) proof) view returns(bool)
func (_Awcontract *AwcontractCallerSession) VerifyProof(proof SparseMerkleTreeProof) (bool, error) {
	return _Awcontract.Contract.VerifyProof(&_Awcontract.CallOpts, proof)
}

// WithdrawalDelay is a free data retrieval call binding the contract method 0xa7ab6961.
//
// Solidity: function withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractCaller) WithdrawalDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "withdrawalDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawalDelay is a free data retrieval call binding the contract method 0xa7ab6961.
//
// Solidity: function withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractSession) WithdrawalDelay() (*big.Int, error) {
	return _Awcontract.Contract.WithdrawalDelay(&_Awcontract.CallOpts)
}

// WithdrawalDelay is a free data retrieval call binding the contract method 0xa7ab6961.
//
// Solidity: function withdrawalDelay() view returns(uint256)
func (_Awcontract *AwcontractCallerSession) WithdrawalDelay() (*big.Int, error) {
	return _Awcontract.Contract.WithdrawalDelay(&_Awcontract.CallOpts)
}

// WithdrawalTime is a free data retrieval call binding the contract method 0x6085dc87.
//
// Solidity: function withdrawalTime(uint256 ) view returns(uint256)
func (_Awcontract *AwcontractCaller) WithdrawalTime(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Awcontract.contract.Call(opts, &out, "withdrawalTime", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawalTime is a free data retrieval call binding the contract method 0x6085dc87.
//
// Solidity: function withdrawalTime(uint256 ) view returns(uint256)
func (_Awcontract *AwcontractSession) WithdrawalTime(arg0 *big.Int) (*big.Int, error) {
	return _Awcontract.Contract.WithdrawalTime(&_Awcontract.CallOpts, arg0)
}

// WithdrawalTime is a free data retrieval call binding the contract method 0x6085dc87.
//
// Solidity: function withdrawalTime(uint256 ) view returns(uint256)
func (_Awcontract *AwcontractCallerSession) WithdrawalTime(arg0 *big.Int) (*big.Int, error) {
	return _Awcontract.Contract.WithdrawalTime(&_Awcontract.CallOpts, arg0)
}

// ClaimWithdrawal is a paid mutator transaction binding the contract method 0xf8444436.
//
// Solidity: function claimWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactor) ClaimWithdrawal(opts *bind.TransactOpts, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "claimWithdrawal", _stakeID)
}

// ClaimWithdrawal is a paid mutator transaction binding the contract method 0xf8444436.
//
// Solidity: function claimWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractSession) ClaimWithdrawal(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.ClaimWithdrawal(&_Awcontract.TransactOpts, _stakeID)
}

// ClaimWithdrawal is a paid mutator transaction binding the contract method 0xf8444436.
//
// Solidity: function claimWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactorSession) ClaimWithdrawal(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.ClaimWithdrawal(&_Awcontract.TransactOpts, _stakeID)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractTransactor) Deposit(opts *bind.TransactOpts, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "deposit", _stakeID)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractSession) Deposit(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Deposit(&_Awcontract.TransactOpts, _stakeID)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractTransactorSession) Deposit(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Deposit(&_Awcontract.TransactOpts, _stakeID)
}

// DepositDebug is a paid mutator transaction binding the contract method 0x85e1538f.
//
// Solidity: function deposit_debug(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractTransactor) DepositDebug(opts *bind.TransactOpts, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "deposit_debug", _stakeID)
}

// DepositDebug is a paid mutator transaction binding the contract method 0x85e1538f.
//
// Solidity: function deposit_debug(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractSession) DepositDebug(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.DepositDebug(&_Awcontract.TransactOpts, _stakeID)
}

// DepositDebug is a paid mutator transaction binding the contract method 0x85e1538f.
//
// Solidity: function deposit_debug(uint256 _stakeID) payable returns()
func (_Awcontract *AwcontractTransactorSession) DepositDebug(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.DepositDebug(&_Awcontract.TransactOpts, _stakeID)
}

// Init is a paid mutator transaction binding the contract method 0x7fae2a5f.
//
// Solidity: function init(uint32 _maxTreeDepth, uint256 _stakeUnit, uint256 _epochLength, uint256 _withdrawalDelay, uint256 _stakeFreezePeriod) returns()
func (_Awcontract *AwcontractTransactor) Init(opts *bind.TransactOpts, _maxTreeDepth uint32, _stakeUnit *big.Int, _epochLength *big.Int, _withdrawalDelay *big.Int, _stakeFreezePeriod *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "init", _maxTreeDepth, _stakeUnit, _epochLength, _withdrawalDelay, _stakeFreezePeriod)
}

// Init is a paid mutator transaction binding the contract method 0x7fae2a5f.
//
// Solidity: function init(uint32 _maxTreeDepth, uint256 _stakeUnit, uint256 _epochLength, uint256 _withdrawalDelay, uint256 _stakeFreezePeriod) returns()
func (_Awcontract *AwcontractSession) Init(_maxTreeDepth uint32, _stakeUnit *big.Int, _epochLength *big.Int, _withdrawalDelay *big.Int, _stakeFreezePeriod *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Init(&_Awcontract.TransactOpts, _maxTreeDepth, _stakeUnit, _epochLength, _withdrawalDelay, _stakeFreezePeriod)
}

// Init is a paid mutator transaction binding the contract method 0x7fae2a5f.
//
// Solidity: function init(uint32 _maxTreeDepth, uint256 _stakeUnit, uint256 _epochLength, uint256 _withdrawalDelay, uint256 _stakeFreezePeriod) returns()
func (_Awcontract *AwcontractTransactorSession) Init(_maxTreeDepth uint32, _stakeUnit *big.Int, _epochLength *big.Int, _withdrawalDelay *big.Int, _stakeFreezePeriod *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Init(&_Awcontract.TransactOpts, _maxTreeDepth, _stakeUnit, _epochLength, _withdrawalDelay, _stakeFreezePeriod)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactor) RequestWithdrawal(opts *bind.TransactOpts, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "requestWithdrawal", _stakeID)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractSession) RequestWithdrawal(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.RequestWithdrawal(&_Awcontract.TransactOpts, _stakeID)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactorSession) RequestWithdrawal(_stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.RequestWithdrawal(&_Awcontract.TransactOpts, _stakeID)
}

// Slash is a paid mutator transaction binding the contract method 0xa22a6428.
//
// Solidity: function slash(uint256 _stakeSecret, uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactor) Slash(opts *bind.TransactOpts, _stakeSecret *big.Int, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.contract.Transact(opts, "slash", _stakeSecret, _stakeID)
}

// Slash is a paid mutator transaction binding the contract method 0xa22a6428.
//
// Solidity: function slash(uint256 _stakeSecret, uint256 _stakeID) returns()
func (_Awcontract *AwcontractSession) Slash(_stakeSecret *big.Int, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Slash(&_Awcontract.TransactOpts, _stakeSecret, _stakeID)
}

// Slash is a paid mutator transaction binding the contract method 0xa22a6428.
//
// Solidity: function slash(uint256 _stakeSecret, uint256 _stakeID) returns()
func (_Awcontract *AwcontractTransactorSession) Slash(_stakeSecret *big.Int, _stakeID *big.Int) (*types.Transaction, error) {
	return _Awcontract.Contract.Slash(&_Awcontract.TransactOpts, _stakeSecret, _stakeID)
}

// AwcontractDebugMsgIterator is returned from FilterDebugMsg and is used to iterate over the raw logs and unpacked data for DebugMsg events raised by the Awcontract contract.
type AwcontractDebugMsgIterator struct {
	Event *AwcontractDebugMsg // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AwcontractDebugMsgIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AwcontractDebugMsg)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AwcontractDebugMsg)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AwcontractDebugMsgIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AwcontractDebugMsgIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AwcontractDebugMsg represents a DebugMsg event raised by the Awcontract contract.
type AwcontractDebugMsg struct {
	Tag string
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDebugMsg is a free log retrieval operation binding the contract event 0x860fab49fdadbdbe734e78cb9bc1a5f504e2d1d356c9930aef6e010448db52b9.
//
// Solidity: event DebugMsg(string tag)
func (_Awcontract *AwcontractFilterer) FilterDebugMsg(opts *bind.FilterOpts) (*AwcontractDebugMsgIterator, error) {

	logs, sub, err := _Awcontract.contract.FilterLogs(opts, "DebugMsg")
	if err != nil {
		return nil, err
	}
	return &AwcontractDebugMsgIterator{contract: _Awcontract.contract, event: "DebugMsg", logs: logs, sub: sub}, nil
}

// WatchDebugMsg is a free log subscription operation binding the contract event 0x860fab49fdadbdbe734e78cb9bc1a5f504e2d1d356c9930aef6e010448db52b9.
//
// Solidity: event DebugMsg(string tag)
func (_Awcontract *AwcontractFilterer) WatchDebugMsg(opts *bind.WatchOpts, sink chan<- *AwcontractDebugMsg) (event.Subscription, error) {

	logs, sub, err := _Awcontract.contract.WatchLogs(opts, "DebugMsg")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AwcontractDebugMsg)
				if err := _Awcontract.contract.UnpackLog(event, "DebugMsg", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDebugMsg is a log parse operation binding the contract event 0x860fab49fdadbdbe734e78cb9bc1a5f504e2d1d356c9930aef6e010448db52b9.
//
// Solidity: event DebugMsg(string tag)
func (_Awcontract *AwcontractFilterer) ParseDebugMsg(log types.Log) (*AwcontractDebugMsg, error) {
	event := new(AwcontractDebugMsg)
	if err := _Awcontract.contract.UnpackLog(event, "DebugMsg", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AwcontractDebugPointIterator is returned from FilterDebugPoint and is used to iterate over the raw logs and unpacked data for DebugPoint events raised by the Awcontract contract.
type AwcontractDebugPointIterator struct {
	Event *AwcontractDebugPoint // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AwcontractDebugPointIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AwcontractDebugPoint)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AwcontractDebugPoint)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AwcontractDebugPointIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AwcontractDebugPointIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AwcontractDebugPoint represents a DebugPoint event raised by the Awcontract contract.
type AwcontractDebugPoint struct {
	Tag string
	V1  *big.Int
	V2  *big.Int
	A   common.Address
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDebugPoint is a free log retrieval operation binding the contract event 0xbc2a4889b9b8fbf4868a3324d5f54510bab360705e6ecc9a9cf609e9e77280a2.
//
// Solidity: event DebugPoint(string tag, uint256 v1, uint256 v2, address a)
func (_Awcontract *AwcontractFilterer) FilterDebugPoint(opts *bind.FilterOpts) (*AwcontractDebugPointIterator, error) {

	logs, sub, err := _Awcontract.contract.FilterLogs(opts, "DebugPoint")
	if err != nil {
		return nil, err
	}
	return &AwcontractDebugPointIterator{contract: _Awcontract.contract, event: "DebugPoint", logs: logs, sub: sub}, nil
}

// WatchDebugPoint is a free log subscription operation binding the contract event 0xbc2a4889b9b8fbf4868a3324d5f54510bab360705e6ecc9a9cf609e9e77280a2.
//
// Solidity: event DebugPoint(string tag, uint256 v1, uint256 v2, address a)
func (_Awcontract *AwcontractFilterer) WatchDebugPoint(opts *bind.WatchOpts, sink chan<- *AwcontractDebugPoint) (event.Subscription, error) {

	logs, sub, err := _Awcontract.contract.WatchLogs(opts, "DebugPoint")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AwcontractDebugPoint)
				if err := _Awcontract.contract.UnpackLog(event, "DebugPoint", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDebugPoint is a log parse operation binding the contract event 0xbc2a4889b9b8fbf4868a3324d5f54510bab360705e6ecc9a9cf609e9e77280a2.
//
// Solidity: event DebugPoint(string tag, uint256 v1, uint256 v2, address a)
func (_Awcontract *AwcontractFilterer) ParseDebugPoint(log types.Log) (*AwcontractDebugPoint, error) {
	event := new(AwcontractDebugPoint)
	if err := _Awcontract.contract.UnpackLog(event, "DebugPoint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
