// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sequencer

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

// SequencerWarpBlock is an auto generated low-level Go binding around an user-defined struct.
type SequencerWarpBlock struct {
	Height     *big.Int
	BlockRoot  *big.Int
	ParentRoot *big.Int
}

// SequencerMetaData contains all meta data concerning the Sequencer contract.
var SequencerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expectedBlockNumber\",\"type\":\"uint256\"}],\"name\":\"IncorrectBlockNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoKeySelected\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEnoughStake\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"numBlocks\",\"type\":\"uint256\"}],\"name\":\"TooManyBlocks\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"NewBlock\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"firstBlockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"numBlocks\",\"type\":\"uint256\"}],\"name\":\"NewBlocks\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockRoot\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentRoot\",\"type\":\"uint256\"}],\"internalType\":\"structSequencer.WarpBlock\",\"name\":\"warp\",\"type\":\"tuple\"}],\"name\":\"addBlock\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"blockHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"}],\"name\":\"commitments\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"commitment\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockRoot\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"parentRoot\",\"type\":\"uint256\"}],\"internalType\":\"structSequencer.WarpBlock[]\",\"name\":\"blocks\",\"type\":\"tuple[]\"}],\"name\":\"newBlocks\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// SequencerABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerMetaData.ABI instead.
var SequencerABI = SequencerMetaData.ABI

// DeploySequencerContract deploys a new Ethereum contract, binding an instance of Sequencer to it.
func DeploySequencerContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Sequencer, error) {
	parsed, err := SequencerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerMetaData.Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Sequencer{SequencerCaller: SequencerCaller{contract: contract}, SequencerTransactor: SequencerTransactor{contract: contract}, SequencerFilterer: SequencerFilterer{contract: contract}}, nil
}

// Sequencer is an auto generated Go binding around an Ethereum contract.
type Sequencer struct {
	SequencerCaller     // Read-only binding to the contract
	SequencerTransactor // Write-only binding to the contract
	SequencerFilterer   // Log filterer for contract events
}

// SequencerCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerSession struct {
	Contract     *Sequencer        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SequencerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerCallerSession struct {
	Contract *SequencerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// SequencerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerTransactorSession struct {
	Contract     *SequencerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SequencerRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerRaw struct {
	Contract *Sequencer // Generic contract binding to access the raw methods on
}

// SequencerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerCallerRaw struct {
	Contract *SequencerCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerTransactorRaw struct {
	Contract *SequencerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencer creates a new instance of Sequencer, bound to a specific deployed contract.
func NewSequencer(address common.Address, backend bind.ContractBackend) (*Sequencer, error) {
	contract, err := bindSequencer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Sequencer{SequencerCaller: SequencerCaller{contract: contract}, SequencerTransactor: SequencerTransactor{contract: contract}, SequencerFilterer: SequencerFilterer{contract: contract}}, nil
}

// NewSequencerCaller creates a new read-only instance of Sequencer, bound to a specific deployed contract.
func NewSequencerCaller(address common.Address, caller bind.ContractCaller) (*SequencerCaller, error) {
	contract, err := bindSequencer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerCaller{contract: contract}, nil
}

// NewSequencerTransactor creates a new write-only instance of Sequencer, bound to a specific deployed contract.
func NewSequencerTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerTransactor, error) {
	contract, err := bindSequencer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerTransactor{contract: contract}, nil
}

// NewSequencerFilterer creates a new log filterer instance of Sequencer, bound to a specific deployed contract.
func NewSequencerFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerFilterer, error) {
	contract, err := bindSequencer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerFilterer{contract: contract}, nil
}

// bindSequencer binds a generic wrapper to an already deployed contract.
func bindSequencer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SequencerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Sequencer *SequencerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Sequencer.Contract.SequencerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Sequencer *SequencerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Sequencer.Contract.SequencerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Sequencer *SequencerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Sequencer.Contract.SequencerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Sequencer *SequencerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Sequencer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Sequencer *SequencerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Sequencer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Sequencer *SequencerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Sequencer.Contract.contract.Transact(opts, method, params...)
}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Sequencer *SequencerCaller) MAXBLOCKS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Sequencer.contract.Call(opts, &out, "MAX_BLOCKS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Sequencer *SequencerSession) MAXBLOCKS() (*big.Int, error) {
	return _Sequencer.Contract.MAXBLOCKS(&_Sequencer.CallOpts)
}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Sequencer *SequencerCallerSession) MAXBLOCKS() (*big.Int, error) {
	return _Sequencer.Contract.MAXBLOCKS(&_Sequencer.CallOpts)
}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Sequencer *SequencerCaller) BlockHeight(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Sequencer.contract.Call(opts, &out, "blockHeight")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Sequencer *SequencerSession) BlockHeight() (*big.Int, error) {
	return _Sequencer.Contract.BlockHeight(&_Sequencer.CallOpts)
}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Sequencer *SequencerCallerSession) BlockHeight() (*big.Int, error) {
	return _Sequencer.Contract.BlockHeight(&_Sequencer.CallOpts)
}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Sequencer *SequencerCaller) Commitments(opts *bind.CallOpts, blockHeight *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Sequencer.contract.Call(opts, &out, "commitments", blockHeight)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Sequencer *SequencerSession) Commitments(blockHeight *big.Int) (*big.Int, error) {
	return _Sequencer.Contract.Commitments(&_Sequencer.CallOpts, blockHeight)
}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Sequencer *SequencerCallerSession) Commitments(blockHeight *big.Int) (*big.Int, error) {
	return _Sequencer.Contract.Commitments(&_Sequencer.CallOpts, blockHeight)
}

// AddBlock is a paid mutator transaction binding the contract method 0x2c1b92b6.
//
// Solidity: function addBlock((uint256,uint256,uint256) warp) returns()
func (_Sequencer *SequencerTransactor) AddBlock(opts *bind.TransactOpts, warp SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.contract.Transact(opts, "addBlock", warp)
}

// AddBlock is a paid mutator transaction binding the contract method 0x2c1b92b6.
//
// Solidity: function addBlock((uint256,uint256,uint256) warp) returns()
func (_Sequencer *SequencerSession) AddBlock(warp SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.Contract.AddBlock(&_Sequencer.TransactOpts, warp)
}

// AddBlock is a paid mutator transaction binding the contract method 0x2c1b92b6.
//
// Solidity: function addBlock((uint256,uint256,uint256) warp) returns()
func (_Sequencer *SequencerTransactorSession) AddBlock(warp SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.Contract.AddBlock(&_Sequencer.TransactOpts, warp)
}

// NewBlocks is a paid mutator transaction binding the contract method 0xff3379f7.
//
// Solidity: function newBlocks((uint256,uint256,uint256)[] blocks) returns()
func (_Sequencer *SequencerTransactor) NewBlocks(opts *bind.TransactOpts, blocks []SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.contract.Transact(opts, "newBlocks", blocks)
}

// NewBlocks is a paid mutator transaction binding the contract method 0xff3379f7.
//
// Solidity: function newBlocks((uint256,uint256,uint256)[] blocks) returns()
func (_Sequencer *SequencerSession) NewBlocks(blocks []SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.Contract.NewBlocks(&_Sequencer.TransactOpts, blocks)
}

// NewBlocks is a paid mutator transaction binding the contract method 0xff3379f7.
//
// Solidity: function newBlocks((uint256,uint256,uint256)[] blocks) returns()
func (_Sequencer *SequencerTransactorSession) NewBlocks(blocks []SequencerWarpBlock) (*types.Transaction, error) {
	return _Sequencer.Contract.NewBlocks(&_Sequencer.TransactOpts, blocks)
}

// SequencerNewBlockIterator is returned from FilterNewBlock and is used to iterate over the raw logs and unpacked data for NewBlock events raised by the Sequencer contract.
type SequencerNewBlockIterator struct {
	Event *SequencerNewBlock // Event containing the contract specifics and raw log

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
func (it *SequencerNewBlockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerNewBlock)
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
		it.Event = new(SequencerNewBlock)
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
func (it *SequencerNewBlockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerNewBlockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerNewBlock represents a NewBlock event raised by the Sequencer contract.
type SequencerNewBlock struct {
	BlockNumber *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterNewBlock is a free log retrieval operation binding the contract event 0x7fe090037171b6c8b269016189ef1438c336d360d819447a441fe06865776049.
//
// Solidity: event NewBlock(uint256 blockNumber)
func (_Sequencer *SequencerFilterer) FilterNewBlock(opts *bind.FilterOpts) (*SequencerNewBlockIterator, error) {

	logs, sub, err := _Sequencer.contract.FilterLogs(opts, "NewBlock")
	if err != nil {
		return nil, err
	}
	return &SequencerNewBlockIterator{contract: _Sequencer.contract, event: "NewBlock", logs: logs, sub: sub}, nil
}

// WatchNewBlock is a free log subscription operation binding the contract event 0x7fe090037171b6c8b269016189ef1438c336d360d819447a441fe06865776049.
//
// Solidity: event NewBlock(uint256 blockNumber)
func (_Sequencer *SequencerFilterer) WatchNewBlock(opts *bind.WatchOpts, sink chan<- *SequencerNewBlock) (event.Subscription, error) {

	logs, sub, err := _Sequencer.contract.WatchLogs(opts, "NewBlock")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerNewBlock)
				if err := _Sequencer.contract.UnpackLog(event, "NewBlock", log); err != nil {
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

// ParseNewBlock is a log parse operation binding the contract event 0x7fe090037171b6c8b269016189ef1438c336d360d819447a441fe06865776049.
//
// Solidity: event NewBlock(uint256 blockNumber)
func (_Sequencer *SequencerFilterer) ParseNewBlock(log types.Log) (*SequencerNewBlock, error) {
	event := new(SequencerNewBlock)
	if err := _Sequencer.contract.UnpackLog(event, "NewBlock", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerNewBlocksIterator is returned from FilterNewBlocks and is used to iterate over the raw logs and unpacked data for NewBlocks events raised by the Sequencer contract.
type SequencerNewBlocksIterator struct {
	Event *SequencerNewBlocks // Event containing the contract specifics and raw log

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
func (it *SequencerNewBlocksIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerNewBlocks)
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
		it.Event = new(SequencerNewBlocks)
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
func (it *SequencerNewBlocksIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerNewBlocksIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerNewBlocks represents a NewBlocks event raised by the Sequencer contract.
type SequencerNewBlocks struct {
	FirstBlockNumber *big.Int
	NumBlocks        *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterNewBlocks is a free log retrieval operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Sequencer *SequencerFilterer) FilterNewBlocks(opts *bind.FilterOpts) (*SequencerNewBlocksIterator, error) {

	logs, sub, err := _Sequencer.contract.FilterLogs(opts, "NewBlocks")
	if err != nil {
		return nil, err
	}
	return &SequencerNewBlocksIterator{contract: _Sequencer.contract, event: "NewBlocks", logs: logs, sub: sub}, nil
}

// WatchNewBlocks is a free log subscription operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Sequencer *SequencerFilterer) WatchNewBlocks(opts *bind.WatchOpts, sink chan<- *SequencerNewBlocks) (event.Subscription, error) {

	logs, sub, err := _Sequencer.contract.WatchLogs(opts, "NewBlocks")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerNewBlocks)
				if err := _Sequencer.contract.UnpackLog(event, "NewBlocks", log); err != nil {
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

// ParseNewBlocks is a log parse operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Sequencer *SequencerFilterer) ParseNewBlocks(log types.Log) (*SequencerNewBlocks, error) {
	event := new(SequencerNewBlocks)
	if err := _Sequencer.contract.UnpackLog(event, "NewBlocks", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
