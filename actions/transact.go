package actions

import (
	"context"
	"errors"
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

var _ chain.Action = (*Transact)(nil)

type Transact struct {
	ContractAddress   ids.ID   `json:"contractAddress"`
	FunctionName      string   `json:"functionName"`
	Input             []byte   `json:"input"` // abi encoded input -> hex to bytes in cmd
	DynamicStateSlots []string `json:"dynamicStateSlots"`
}

func (*Transact) GetTypeID() uint8 {
	return transactID
}

func (t *Transact) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	stateKeys := state.Keys{
		string(storage.ContractKey(t.ContractAddress)): state.Read,
	}
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
	for _, v := range t.DynamicStateSlots {
		p.PackString(v)
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
	p.UnpackBytes(-1, false, &transact.Input) // TODO: try and limit it to a certain size. i.e max 128 KiB
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
	ctxWasm := context.Background()

	contractBytes, err := storage.GetContract(ctx, mu, contractAddress)
	if err != nil {
		return nil, errors.New("contract not deployed")
	}
	err = wasm.Runtime(ctx, ctxWasm, mu, timeStamp, contractAddress, actor, function, contractBytes, inputBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (*Transact) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Transact) UseFeeMarket() bool {
	return true
}
