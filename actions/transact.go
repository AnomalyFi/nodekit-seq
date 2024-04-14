package actions

import (
	"context"
	"errors"
	"strconv"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/anomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
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
	return TransactID
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

func (*Transact) MaxComputeUnits(chain.Rules) uint64 {
	return TransactMaxComputeUnits
}

func (t *Transact) Size() int {
	return codec.StringLen(t.FunctionName) + codec.BytesLen(t.Input) + consts.IDLen
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
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	_ ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	function := t.FunctionName
	inputBytes := t.Input
	contractAddress := t.ContractAddress

	var allocate_ptr api.Function
	var gasUsed uint64 = BaseComputeUnits
	var hasEncError bool
	deployedCodeAtContractAddress, err := storage.GetContract(ctx, mu, contractAddress)
	if err != nil {
		return false, gasUsed, []byte("Contract not deployed"), nil, nil
	}

	ctxWasm := context.Background()

	r := wazero.NewRuntime(ctxWasm)
	defer r.Close(ctxWasm)

	compiledMod, err := r.CompileModule(ctxWasm, deployedCodeAtContractAddress)

	if err != nil {
		return false, gasUsed, utils.ErrBytes(errors.New("couldnot compile contract")), nil, nil
	}

	expFunc := compiledMod.ExportedFunctions()
	checkFunc := expFunc[function]

	if checkFunc == nil {
		return false, gasUsed, utils.ErrBytes(errors.New("invalid function signature")), nil, nil
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
	_, err = r.NewHostModuleBuilder("env").NewFunctionBuilder().
		WithFunc(stateGetBytesInner).Export("stateGetBytes").
		NewFunctionBuilder().WithFunc(stateStoreBytesInner).Export("stateStoreBytes").
		NewFunctionBuilder().WithFunc(stateStoreDynamicBytesInner).Export("stateStoreDynamicBytes").
		NewFunctionBuilder().WithFunc(stateGetDynamicBytesInner).Export("stateGetDynamicBytes").
		Instantiate(ctxWasm)

	if err != nil {
		return false, gasUsed, utils.ErrBytes(err), nil, nil
	}

	mod, err := r.Instantiate(ctxWasm, deployedCodeAtContractAddress)
	if err != nil {
		return false, gasUsed, utils.ErrBytes(err), nil, nil
	}
	allocate_ptr = mod.ExportedFunction("allocate_ptr")
	deallocate_ptr := mod.ExportedFunction("deallocate_ptr")
	txFunction := mod.ExportedFunction(function)

	inputBytesLen := uint64(len(inputBytes))
	results, err := allocate_ptr.Call(ctxWasm, inputBytesLen)
	if err != nil {
		return false, gasUsed, utils.ErrBytes(err), nil, nil
	}
	inputPtr := results[0]
	defer deallocate_ptr.Call(ctxWasm, inputPtr, inputBytesLen) //@todo end abruptly or let dereferec?

	mod.Memory().Write(uint32(inputPtr), inputBytes) // This shold not fail

	//@todo experiment with output
	txResult, err := txFunction.Call(ctxWasm, inputPtr, inputBytesLen)
	//@todo introduce call back mechanism for every function. Add access control for callback mechansisms and
	if err != nil {
		return false, gasUsed, utils.ErrBytes(err), nil, nil
	}
	result := txResult[0]
	if result == 1 {
		return true, gasUsed, nil, nil, nil
	}
	hasEncError = false
	return hasEncError, gasUsed, []byte{byte(result)}, nil, nil
}
