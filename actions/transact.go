package actions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/storage"
	sp1 "github.com/AnomalyFi/sp1-recursion-gnark/sp1"
	babybear "github.com/AnomalyFi/sp1-recursion-gnark/sp1/babybear"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var _ chain.Action = (*Transact)(nil)

type Transact struct {
	FunctionName    string `json:"functionName"`
	ContractAddress ids.ID `json:"contractAddress"`
	Input           []byte `json:"input"` // abi encoded input -> hex to bytes in cmd
}

func (*Transact) GetTypeID() uint8 {
	return transactID
}

func (t *Transact) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	stateKeys := state.Keys{
		string(storage.ContractKey(t.ContractAddress)): state.Read,
	}
	for i := 0; i < consts.NumStateKeys; i++ {
		keyName := "slot" + strconv.Itoa(i)
		stateKeys.Add(string(storage.StateStorageKey(t.ContractAddress, keyName)), state.Read|state.Write)
	}
	return stateKeys
}

func (*Transact) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.BalanceChunks, storage.BalanceChunks} //@todo tune this /change this
}

func (t *Transact) Marshal(p *codec.Packer) {
	p.PackString(t.FunctionName)
	p.PackID(t.ContractAddress)
	p.PackBytes(t.Input)
}

func (*Transact) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return TransactComputeUnits
}

func (t *Transact) Size() int {
	return codec.StringLen(t.FunctionName) + codec.BytesLen(t.Input) + ids.IDLen
}

func (*Transact) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func UnmarshalTransact(p *codec.Packer) (chain.Action, error) {
	var transact Transact
	transact.FunctionName = p.UnpackString(true)
	p.UnpackID(true, &transact.ContractAddress)
	p.UnpackBytes(1024, false, &transact.Input)

	return &transact, nil
}

func (t *Transact) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
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

	// get bytes at state at dynamic slot i
	stateGetDynamicBytesInner := func(ctxInner context.Context, m api.Module, offset uint32, key uint32) uint64 {
		i := 128 + (offset*key)%896
		slot := "slot" + strconv.Itoa(int(i))
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset2 := results[0]
		m.Memory().Write(uint32(offset2), result)
		return uint64(offset2)<<32 | size
	}

	// store bytes in state at dynamic slot i
	stateStoreDynamicBytesInner := func(ctxInner context.Context, m api.Module, offset uint32, key uint32, ptr uint32, size uint32) {
		i := 128 + (offset*key)%896
		slot := "slot" + strconv.Itoa(int(i))
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			hasEncError = true // handle error properly
		} else {
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}

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
		sp1Circuit := sp1.Circuit{
			Vars:                 []frontend.Variable{},
			Felts:                []babybear.Variable{},
			Exts:                 []babybear.ExtensionVariable{},
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
	// fallbacks
	addBalance := func(ctxInner context.Context, m api.Module) {}
	subBalance := func(ctxInner context.Context, m api.Module) {}

	_, err = r.NewHostModuleBuilder("env").NewFunctionBuilder().
		WithFunc(stateGetBytesInner).Export("stateGetBytes").
		NewFunctionBuilder().WithFunc(stateStoreBytesInner).Export("stateStoreBytes").
		NewFunctionBuilder().WithFunc(stateStoreDynamicBytesInner).Export("stateStoreDynamicBytes").
		NewFunctionBuilder().WithFunc(stateGetDynamicBytesInner).Export("stateGetDynamicBytes").
		NewFunctionBuilder().WithFunc(gnarkVerify).Export("gnarkVerify").
		NewFunctionBuilder().WithFunc(addBalance).Export("addBalance").
		NewFunctionBuilder().WithFunc(subBalance).Export("subBalance").
		Instantiate(ctxWasm)

	if err != nil {
		return nil, fmt.Errorf("can not build host module: %s", err.Error())
	}

	// Instantiate the module
	mod, err := r.Instantiate(ctxWasm, deployedCodeAtContractAddress)
	if err != nil {
		return nil, fmt.Errorf("can not instantiate: %s", err.Error())
	}

	// Get the exported functions
	allocate_ptr = mod.ExportedFunction("allocate_ptr")
	deallocate_ptr := mod.ExportedFunction("deallocate_ptr")
	txFunction := mod.ExportedFunction(function)

	// allocate memory for input
	inputBytesLen := uint64(len(inputBytes))
	results, err := allocate_ptr.Call(ctxWasm, inputBytesLen)
	if err != nil {
		return nil, fmt.Errorf("can not allocate memory: %s", err.Error())
	}
	inputPtr := results[0]
	defer deallocate_ptr.Call(ctxWasm, inputPtr, inputBytesLen)
	mod.Memory().Write(uint32(inputPtr), inputBytes) // This shold not fail

	//@todo experiment with output
	txResult, err := txFunction.Call(ctxWasm, inputPtr, inputBytesLen)
	//@todo introduce call back mechanism for every function. Add access control for callback mechansisms and
	if err != nil {
		return nil, fmt.Errorf("can not call tx function: %s", err.Error())
	}
	result := txResult[0]
	if result == 1 && !hasEncError {
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
