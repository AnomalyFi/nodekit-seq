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
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/succinctlabs/gnark-plonky2-verifier/poseidon"
	"github.com/succinctlabs/gnark-plonky2-verifier/variables"
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
	for i := 0; i < NumStateKeys; i++ {
		keyName := "slot" + strconv.Itoa(i)
		stateKeys.Add(string(storage.StateStorageKey(t.ContractAddress, keyName)), state.Read|state.Write)
	}
	return stateKeys
}

func (*Transact) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.BalanceChunks, storage.BalanceChunks} //@todo tune this /change this
}

func (*Transact) OutputsWarpMessage() bool {
	return false
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

func UnmarshalTransact(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
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
		return nil, errors.New("couldnot compile contract")
	}

	expFunc := compiledMod.ExportedFunctions()
	checkFunc := expFunc[function]

	if checkFunc == nil {
		return nil, errors.New("invalid function signature")
	}

	stateGetBytesInner := func(ctxInner context.Context, m api.Module, i uint32) uint64 {
		slot := "slot" + strconv.Itoa(int(i))
		result, _ := storage.GetBytes(ctx, mu, contractAddress, slot)
		size := uint64(len(result))
		results, _ := allocate_ptr.Call(ctxInner, size)
		offset := results[0]
		m.Memory().Write(uint32(offset), result)
		return uint64(offset)<<32 | size
	}
	stateStoreBytesInner := func(ctxInner context.Context, m api.Module, i uint32, ptr uint32, size uint32) {
		slot := "slot" + strconv.Itoa(int(i))
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			hasEncError = true // handle error properly
		} else {
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}
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
	stateStoreDynamicBytesInner := func(ctxInner context.Context, m api.Module, offset uint32, key uint32, ptr uint32, size uint32) {
		i := 128 + (offset*key)%896
		slot := "slot" + strconv.Itoa(int(i))
		if bytes, ok := m.Memory().Read(ptr, size); !ok {
			hasEncError = true // handle error properly
		} else {
			storage.SetBytes(ctx, mu, contractAddress, slot, bytes)
		}
	}
	// precompiles
	gnarkVerify := func(ctxInner context.Context, m api.Module, ptr uint32, size uint32) uint32 {
		// fetch vk, abi from cache(these should exist)
		rHA, _ := rules.FetchCustom("")
		helper := rHA.(genesis.RulesHelper)
		vk := helper.VerificationKey
		abi := helper.GnarkPrecompileABI
		// read from memory
		dataBytes, ok := m.Memory().Read(ptr, size)
		if !ok {
			return 0
		}
		// abi unpack
		method := abi.Methods["gnarkPrecompileInputsDummyFunction"]
		upack, err := method.Inputs.Unpack(dataBytes)
		if err != nil {
			return 0
		}
		input := upack[0].(*GnarkPrecompileInputs)
		// build digest from digest big int
		digest, ok := (*helper.FunctionIDBigIntCache)[input.FunctionIdBigInt]
		if !ok {
			cirucitDigest := frontend.Variable(input.FunctionIdBigInt)
			digestInner := poseidon.BN254HashOut(cirucitDigest)
			(*helper.FunctionIDBigIntCache)[input.FunctionIdBigInt] = digestInner
			digest = digestInner
		}
		// build proof
		proof := plonk.NewProof(ecc.BN254)
		_, err = proof.ReadFrom(bytes.NewBuffer(input.Proof))
		if err != nil {
			// ill-structured proof
			return 0
		}
		// prepare input and output for verification
		inputHash := sha256.Sum256(input.Input)
		outputHash := sha256.Sum256(input.Output)
		inputHashB := new(big.Int).SetBytes(inputHash[:])
		outputHashB := new(big.Int).SetBytes(outputHash[:])
		inputHashM := new(big.Int).And(inputHashB, mask)
		outputHashM := new(big.Int).And(outputHashB, mask)
		if inputHashM.BitLen() > 253 || outputHashM.BitLen() > 253 {
			// ill-structured input or output
			return 0
		}
		assg := &Plonky2xVerifierCircuit{
			VerifierDigest: digest,
			InputHash:      inputHashM,
			OutputHash:     outputHashM,
			ProofWithPis:   variables.ProofWithPublicInputs{},
			VerifierData:   variables.VerifierOnlyCircuitData{},
		}
		wit, err := frontend.NewWitness(assg, ecc.BN254.ScalarField())
		if err != nil {
			// this usually will not happen
			return 0
		}
		pubWit, err := wit.Public()
		if err != nil {
			// this usually will not happend
			return 0
		}
		err = plonk.Verify(proof, vk, pubWit)
		if err != nil {
			// wrong proof - pubWit - vk pair or malicious proof
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

	mod, err := r.Instantiate(ctxWasm, deployedCodeAtContractAddress)
	if err != nil {
		return nil, fmt.Errorf("can not instantiate: %s", err.Error())
	}
	allocate_ptr = mod.ExportedFunction("allocate_ptr")
	deallocate_ptr := mod.ExportedFunction("deallocate_ptr")
	txFunction := mod.ExportedFunction(function)

	inputBytesLen := uint64(len(inputBytes))
	results, err := allocate_ptr.Call(ctxWasm, inputBytesLen)
	if err != nil {
		return nil, fmt.Errorf("can not allocate memory: %s", err.Error())
	}
	inputPtr := results[0]
	defer deallocate_ptr.Call(ctxWasm, inputPtr, inputBytesLen) //@todo end abruptly or let dereferec?

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
