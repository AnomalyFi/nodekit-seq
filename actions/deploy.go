package actions

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	nconsts "github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/storage"
	wasm "github.com/AnomalyFi/nodekit-seq/wasm"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*Deploy)(nil)

type Deploy struct {
	// WASM Bytecode of Smart contracts. This code will be stored in the state of the chain, with contractID as txID of the deploy transaction.
	// For compiling SEQ compatible WASM Bytecode, check https://github.com/AnomalyFi/seq-wasm/tree/main/sdk
	ContractCode []byte `json:"contractCode"`
	// Function called after the contract is deployed. This function will be called with the input provided in the deploy transaction.
	// The function should be present in the contract code. check https://github.com/AnomalyFi/seq-wasm/tree/main/sdk#usage
	InitializerFunctionName string `json:"initializerFunctionName"`
	// Input to the initializer function. This input will be passed to the initializer function.
	Input []byte `json:"input"`
	// Non-default storage slots touched by Initializer function execution.
	DynamicStateSlots []string `json:"dynamicStateSlots"`
}

func (*Deploy) GetTypeID() uint8 {
	return deployID
}

// StateKeys returns the keys that the action reads or writes or allocates.
// 128 static keys from slot0 to slot127 are allocated for the contract.
// If a contract needs more keys or want to implement dynamic types like mapping or an array, it can use dynamic keys.
// Check: https://github.com/AnomalyFi/seq-wasm/tree/main/sdk#state
func (d *Deploy) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	stateKeys := state.Keys{
		string(storage.ContractKey(txID)): state.All,
	}
	for i := 0; i < nconsts.NumStaticStateKeys; i++ {
		keyName := "slot" + strconv.Itoa(i)
		stateKeys.Add(string(storage.StateStorageKey(txID, keyName)), state.All)
	}
	for _, v := range d.DynamicStateSlots {
		stateKeys.Add(string(storage.StateStorageKey(txID, v)), state.All)
	}
	return stateKeys
}

func (d *Deploy) StateKeysMaxChunks() []uint16 {
	chunks := []uint16{consts.MaxUint16}
	for i := 0; i < nconsts.NumStaticStateKeys+len(d.DynamicStateSlots); i++ {
		chunks = append(chunks, storage.StateChunks)
	}
	return chunks
}

func (d *Deploy) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	timeStamp int64,
	actor codec.Address,
	txID ids.ID,
) ([][]byte, error) {
	if !IsWhiteListed(rules, actor) {
		return nil, ErrNotWhiteListed
	}

	if err := storage.SetContract(ctx, mu, txID, d.ContractCode); err != nil {
		return nil, fmt.Errorf("error storing contract: %s", err)
	}

	function := d.InitializerFunctionName
	inputBytes := d.Input
	contractAddress := txID
	ctxWasm := context.Background()

	err := wasm.Runtime(ctx, ctxWasm, mu, timeStamp, contractAddress, actor, function, d.ContractCode, inputBytes)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (*Deploy) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return DeployComputeUnits
}

func (d *Deploy) Marshal(p *codec.Packer) {
	p.PackBytes(d.ContractCode)
	p.PackString(d.InitializerFunctionName)
	p.PackBytes(d.Input)
	strArrLen := len(d.DynamicStateSlots)
	p.PackInt(strArrLen)
	for _, v := range d.DynamicStateSlots {
		p.PackString(v)
	}
}

func UnmarshalDeploy(p *codec.Packer) (chain.Action, error) {
	var deploy Deploy
	p.UnpackBytes(int(math.MaxUint32), true, &deploy.ContractCode)
	deploy.InitializerFunctionName = p.UnpackString(false)
	p.UnpackBytes(-1, false, &deploy.Input)
	strArrLen := p.UnpackInt(false)
	for i := 0; i < strArrLen; i++ {
		deploy.DynamicStateSlots = append(deploy.DynamicStateSlots, p.UnpackString(true))
	}
	return &deploy, p.Err()
}

func (d *Deploy) Size() int {
	var l int
	for _, v := range d.DynamicStateSlots {
		l += codec.StringLen(v)
	}
	return l + codec.BytesLen(d.ContractCode) + codec.BytesLen(d.Input) + codec.StringLen(d.InitializerFunctionName)
}

func (*Deploy) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*Deploy) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Deploy) UseFeeMarket() bool {
	return true
}
