package actions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	nconsts "github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var _ chain.Action = (*Transact)(nil)

type Transact struct {
	ContractAddress ids.ID `json:"contractAddress"`
	FunctionName    string `json:"functionName"`
	Input           []byte `json:"input"` // abi encoded input -> hex to bytes in cmd
	// dynamic state slots?
	DynamicStateSlots []string `json:"dynamicStateSlots"`
}

func (*Transact) GetTypeID() uint8 {
	return transactID
}

func (t *Transact) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	stateKeys := state.Keys{
		string(storage.ContractKey(t.ContractAddress)): state.Read,
	}
	// do not limit state keys to have slot + integer concatenated
	for i := 0; i < nconsts.NumStaticStateKeys; i++ {
		keyName := "slot" + strconv.Itoa(i)
		stateKeys.Add(string(storage.StateStorageKey(t.ContractAddress, keyName)), state.All)
	}
	for _, v := range t.DynamicStateSlots {
		stateKeys.Add(string(storage.StateStorageKey(t.ContractAddress, v)), state.All)
	}
	return stateKeys
}

func (t *Transact) StateKeysMaxChunks() []uint16 {
	chunks := []uint16{consts.MaxUint16}
	for i := 0; i < nconsts.NumStaticStateKeys+len(t.DynamicStateSlots); i++ {
		chunks = append(chunks, storage.StateChunks)
	}
	return chunks
}

func (t *Transact) Marshal(p *codec.Packer) {
	p.PackString(t.FunctionName)
	p.PackID(t.ContractAddress)
	p.PackBytes(t.Input)
	strArrLen := len(t.DynamicStateSlots)
	p.PackInt(strArrLen)
	for i := 0; i < strArrLen; i++ {
		p.PackString(t.DynamicStateSlots[i])
	}
}

func (*Transact) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return TransactComputeUnits
}

func (t *Transact) Size() int {
	var l int
	for _, v := range t.DynamicStateSlots {
		l += codec.StringLen(v)
	}
	return codec.StringLen(t.FunctionName) + codec.BytesLen(t.Input) + ids.IDLen + consts.IntLen + l
}

func (*Transact) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func UnmarshalTransact(p *codec.Packer) (chain.Action, error) {
	var transact Transact
	transact.FunctionName = p.UnpackString(true)
	p.UnpackID(true, &transact.ContractAddress)
	p.UnpackBytes(-1, false, &transact.Input) // @todo try and limit it to a certain size. i.e max 128 KiB
	// unpack dynamic state storage slots.
	strArrLen := p.UnpackInt(false)
	for i := 0; i < strArrLen; i++ {
		transact.DynamicStateSlots = append(transact.DynamicStateSlots, p.UnpackString(true))
	}
	return &transact, nil
}

func (t *Transact) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	timeStamp int64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	function := t.FunctionName
	inputBytes := t.Input
	contractAddress := t.ContractAddress

	var allocate_ptr api.Function
	var hasEncError bool
	deployedCodeAtContractAddress, err := storage.GetContract(ctx, mu, contractAddress)
	if err != nil {
		return nil, errors.New("contract not deployed")
	}

	ctxWasm := context.Background()

	r := wazero.NewRuntime(ctxWasm)
	defer r.Close(ctxWasm)

	compiledMod, err := r.CompileModule(ctxWasm, deployedCodeAtContractAddress)
	if err != nil {
		return nil, errors.New("contract not compiled")
	}

	expFunc := compiledMod.ExportedFunctions()
	checkFunc := expFunc[function]
	if checkFunc == nil {
		return nil, errors.New("invalid function signature")
	}

	// System calls

	// get bytes from state at slot i
	stateGetBytesInner := func(ctxInner context.Context, m api.Module, i uint32) uint64 {
		slot := "slot" + strconv.Itoa(int(i))
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset := results[0]
		m.Memory().Write(uint32(offset), result)
		return uint64(offset)<<32 | size
	}

	// store bytes in state at slot i
	stateStoreBytesInner := func(ctxInner context.Context, m api.Module, i uint32, ptr uint32, size uint32) {
		slot := "slot" + strconv.Itoa(int(i))
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			hasEncError = true // handle error properly
		} else {
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}

	// store bytes in state at dynamic slot i
	stateStoreDynamicBytesInner := func(ctxInner context.Context, m api.Module, id, ptrKey, sizeOfKey, ptr, size uint32) {
		// read key from memory.
		key, ok := m.Memory().Read(ptrKey, sizeOfKey)
		if !ok {
			hasEncError = true // handle error properly
		}
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			hasEncError = true // handle error properly
		} else {
			slot := "slot" + strconv.Itoa(int(id)) + hex.EncodeToString(key)
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}

	// get bytes at state at dynamic slot i
	stateGetDynamicBytesInner := func(ctxInner context.Context, m api.Module, id, ptrKey, sizeOfKey uint32) uint64 {
		// read key from memory.
		key, ok := m.Memory().Read(ptrKey, sizeOfKey)
		if !ok {
			os.Exit(10)
		}
		slot := "slot" + strconv.Itoa(int(id)) + hex.EncodeToString(key)
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		// write value to memory.
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset2 := results[0]
		m.Memory().Write(uint32(offset2), result)
		return uint64(offset2)<<32 | size
	}

	// Precompiles
	// SP1 plonk proof verifier pre-compile
	gnarkVerify := func(ctxInner context.Context, m api.Module, ptr uint32, size uint32) uint32 {
		// read from memory
		dataBytes, ok := m.Memory().Read(ptr, size)
		if !ok {
			return 0
		}
		// abi unpack the data
		method := GnarkPreCompileABI.Methods["gnarkPrecompile"]
		upack, err := method.Inputs.Unpack(dataBytes)
		if err != nil {
			return 0
		}

		// calculate publicValuesDisgest
		preCompileInput := upack[0].(*GnarkPrecompileInputs)
		publicValuesHash := sha256.Sum256(preCompileInput.PublicValues)
		publicValuesB := new(big.Int).SetBytes(publicValuesHash[:])
		publicValuesDigest := new(big.Int).And(publicValuesB, mask)
		if publicValuesDigest.BitLen() > 253 {
			return 0
		}
		sp1Circuit := SP1Circuit{
			Vars:                 []frontend.Variable{},
			Felts:                []babybearVariable{},
			Exts:                 []babybearExtensionVariable{},
			VkeyHash:             preCompileInput.ProgramVKey,
			CommitedValuesDigest: publicValuesDigest,
		}

		// read vk from preCompileInput
		vk := plonk.NewVerifyingKey(ecc.BN254)
		_, err = vk.ReadFrom(bytes.NewBuffer(preCompileInput.ProgramVKey))
		if err != nil {
			fmt.Printf("failed to read vk file: %s", err)
			return 0
		}

		// read proof from preCompileInput
		proof := plonk.NewProof(ecc.BN254)
		_, err = proof.ReadFrom(bytes.NewBuffer(preCompileInput.ProofBytes))
		if err != nil {
			fmt.Println(err)
			return 0
		}

		// create witness
		wit, err := frontend.NewWitness(&sp1Circuit, ecc.BN254.ScalarField())
		if err != nil {
			return 0
		}

		// get the public witness
		pubWit, err := wit.Public()
		if err != nil {
			return 0
		}

		// verify the proof
		err = plonk.Verify(proof, vk, pubWit)
		if err != nil {
			// the vk may not be corresponding to the proof or public witness are not corresponding to proofs or proof is invalid
			return 0
		}
		return 1
	}

	setBalance := func(ctxInner context.Context, m api.Module, addressPtr, assetPtr uint32, amount uint64) uint32 {
		addrBytes, ok := m.Memory().Read(addressPtr, codec.AddressLen)
		if !ok {
			return 0
		}
		assetBytes, ok := m.Memory().Read(assetPtr, codec.AddressLen)
		if !ok {
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
			return 0
		}
		assetBytes, ok := m.Memory().Read(assetPtr, codec.AddressLen)
		if !ok {
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
		return nil, fmt.Errorf("error building host module env: %s", err.Error())
	}

	// build new host module "precompiles"
	_, err = r.NewHostModuleBuilder("precompiles").
		NewFunctionBuilder().WithFunc(gnarkVerify).Export("gnarkVerify").
		NewFunctionBuilder().WithFunc(setBalance).Export("setBalance").
		NewFunctionBuilder().WithFunc(getBalance).Export("getBalance").
		Instantiate(ctxWasm)
	if err != nil {
		return nil, fmt.Errorf("error building host module precompiles: %s", err.Error())
	}

	// Instantiate the module
	mod, err := r.Instantiate(ctxWasm, deployedCodeAtContractAddress)
	if err != nil {
		return nil, fmt.Errorf("error instantiating: %s", err.Error())
	}

	// Get the exported functions
	allocate_ptr = mod.ExportedFunction("allocate_ptr")
	deallocate_ptr := mod.ExportedFunction("deallocate_ptr")
	txFunction := mod.ExportedFunction(function)

	// Allocate and write to memory message sender and tx context.
	results, err := allocate_ptr.Call(ctxWasm, codec.AddressLen)
	if err != nil {
		return nil, fmt.Errorf("error allocating memory: %s", err.Error())
	}
	address_ptr := results[0]
	defer deallocate_ptr.Call(ctxWasm, address_ptr, codec.AddressLen)
	mod.Memory().Write(uint32(address_ptr), actor[:])

	txContext := TxContext{timestamp: timeStamp, msgSenderPtr: uint32(results[0])}
	txContextBytes := txContextToBytes(txContext)

	results, err = allocate_ptr.Call(ctxWasm, uint64(len(txContextBytes)))
	if err != nil {
		return nil, fmt.Errorf("error allocating memory: %s", err.Error())
	}
	txContextPtr := results[0]
	defer deallocate_ptr.Call(ctxWasm, txContextPtr, uint64(len(txContextBytes)))
	mod.Memory().Write(uint32(txContextPtr), txContextBytes)

	// allocate memory for input
	inputBytesLen := uint64(len(inputBytes))
	results, err = allocate_ptr.Call(ctxWasm, inputBytesLen)
	if err != nil {
		return nil, fmt.Errorf("error allocating memory: %s", err.Error())
	}
	inputPtr := results[0]
	defer deallocate_ptr.Call(ctxWasm, inputPtr, inputBytesLen)
	mod.Memory().Write(uint32(inputPtr), inputBytes)

	// call the function.
	txResult, err := txFunction.Call(ctxWasm, txContextPtr, inputPtr, inputBytesLen)
	if err != nil {
		return nil, fmt.Errorf("error calling function: %s", err.Error())
	}
	// value returned = 1, function call is successful. otherwise function call is not successfull, revert state changes if any.
	if txResult[0] == 1 && !hasEncError {
		return nil, nil
	}
	return nil, errors.New("error in contract execution")
}

func (*Transact) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Transact) UseFeeMarket() bool {
	return false
}
