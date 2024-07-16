package wasm

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/wasm/precompiles"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func Runtime(
	ctx context.Context,
	ctxWasm context.Context,
	mu state.Mutable,
	timeStamp int64,
	contractAddress ids.ID,
	actor codec.Address,
	function string,
	contractBytes []byte,
	inputBytes []byte,
) error {

	r := wazero.NewRuntime(ctxWasm)
	var allocate_ptr api.Function
	defer r.Close(ctxWasm)

	compiledMod, err := r.CompileModule(ctxWasm, contractBytes)
	if err != nil {
		return ErrContractCompile
	}

	expFunc := compiledMod.ExportedFunctions()
	checkFunc := expFunc[function]
	if checkFunc == nil {
		return ErrInvalidFuncSig
	}

	/// System calls

	// store bytes in state at slot i
	stateStoreBytesInner := func(ctxInner context.Context, m api.Module, i uint32, ptr uint32, size uint32) {
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			// TODO
			return
		} else {
			slot := binary.BigEndian.AppendUint32(nil, i)
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}

	// get bytes from state at slot i
	stateGetBytesInner := func(ctxInner context.Context, m api.Module, i uint32) uint64 {
		slot := binary.BigEndian.AppendUint32(nil, i)
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset := results[0]
		m.Memory().Write(uint32(offset), result)
		return uint64(offset)<<32 | size
	}

	// store bytes in state at dynamic slot i
	stateStoreDynamicBytesInner := func(ctxInner context.Context, m api.Module, id, ptrKey, sizeOfKey, ptr, size uint32) {
		// read key from memory.
		key, ok := m.Memory().Read(ptrKey, sizeOfKey)
		if !ok {
			// TODO
			return
		}
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			// TODO
			return
		} else {
			slot := append(binary.BigEndian.AppendUint32(nil, id), key...)
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}

	// get bytes at state at dynamic slot i
	stateGetDynamicBytesInner := func(ctxInner context.Context, m api.Module, id, ptrKey, sizeOfKey uint32) uint64 {
		// read key from memory.
		key, ok := m.Memory().Read(ptrKey, sizeOfKey)
		if !ok {
			// TODO
			return 0
		}
		slot := append(binary.BigEndian.AppendUint32(nil, id), key...)
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		// write value to memory.
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset2 := results[0]
		m.Memory().Write(uint32(offset2), result)
		return uint64(offset2)<<32 | size
	}

	setBalance := func(ctxInner context.Context, m api.Module, addressPtr, assetPtr uint32, amount uint64) uint32 {
		addrBytes, ok := m.Memory().Read(addressPtr, codec.AddressLen)
		if !ok {
			// TODO:
			return 0
		}
		assetBytes, ok := m.Memory().Read(assetPtr, codec.AddressLen)
		if !ok {
			// TODO:
			return 0
		}
		addr := codec.Address(addrBytes)
		asset := ids.ID(assetBytes)
		if err := storage.SetBalance(ctx, mu, addr, asset, amount); err != nil {
			return 0
		}
		return 1
	}

	getBalance := func(ctxInner context.Context, m api.Module, addressPtr, assetPtr uint32) uint64 {
		addrBytes, ok := m.Memory().Read(addressPtr, codec.AddressLen)
		if !ok {
			// TODO:
			return 0
		}
		assetBytes, ok := m.Memory().Read(assetPtr, codec.AddressLen)
		if !ok {
			// TODO:
			return 0
		}
		addr := codec.Address(addrBytes)
		asset := ids.ID(assetBytes)
		balance, err := storage.GetBalance(ctx, mu, addr, asset)
		if err != nil {
			return 0
		}
		return balance
	}

	// build new host module "env"
	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(stateGetBytesInner).Export("stateGetBytes").
		NewFunctionBuilder().WithFunc(stateStoreBytesInner).Export("stateStoreBytes").
		NewFunctionBuilder().WithFunc(stateStoreDynamicBytesInner).Export("stateStoreDynamicBytes").
		NewFunctionBuilder().WithFunc(stateGetDynamicBytesInner).Export("stateGetDynamicBytes").
		Instantiate(ctxWasm)
	if err != nil {
		return fmt.Errorf("error building host module env: %s", err.Error())
	}

	// build new host module "precompiles"
	_, err = r.NewHostModuleBuilder("precompiles").
		NewFunctionBuilder().WithFunc(precompiles.GnarkPreCompile).Export("gnarkVerify").
		NewFunctionBuilder().WithFunc(setBalance).Export("setBalance").
		NewFunctionBuilder().WithFunc(getBalance).Export("getBalance").
		Instantiate(ctxWasm)
	if err != nil {
		return fmt.Errorf("error building host module precompiles: %s", err.Error())
	}

	// Instantiate the module
	mod, err := r.Instantiate(ctxWasm, contractBytes)
	if err != nil {
		return fmt.Errorf("error instantiating: %s", err.Error())
	}
	// Get the exported functions
	allocate_ptr = mod.ExportedFunction("allocate_ptr")
	// deallocate_ptr := mod.ExportedFunction("deallocate_ptr")
	txFunction := mod.ExportedFunction(function)

	// Allocate and write to memory message sender and tx context.
	results, err := allocate_ptr.Call(ctxWasm, codec.AddressLen+uint64(len(inputBytes))+12 /*txContext*/)
	if err != nil {
		return fmt.Errorf("error allocating memory: %s", err.Error())
	}
	address_ptr := results[0]
	mod.Memory().Write(uint32(address_ptr), actor[:])

	txContext := TxContext{timestamp: timeStamp, msgSenderPtr: uint32(results[0])}
	txContextBytes := txContextToBytes(txContext)

	txContextPtr := address_ptr + codec.AddressLen
	mod.Memory().Write(uint32(txContextPtr), txContextBytes)

	// allocate memory for input
	inputBytesLen := uint64(len(inputBytes))
	inputPtr := txContextPtr + uint64(len(txContextBytes))
	mod.Memory().Write(uint32(inputPtr), inputBytes)

	// call the function.
	txResult, err := txFunction.Call(ctxWasm, txContextPtr, inputPtr, inputBytesLen)
	if err != nil {
		return fmt.Errorf("error calling function: %s", err.Error())
	}
	// value returned = 1, function call is successful. otherwise function call is not successfull, revert state changes if any.
	if txResult[0] == 1 {
		return nil
	}
	return ErrExecutionRevert
}
