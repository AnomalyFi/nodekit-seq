package actions

import (
	"context"
	"fmt"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*ArcadiaRegister)(nil)

type ArcadiaRegister struct {
	Namespace  []byte `json:"namespace"`
	StartEpoch uint64 `json:"start_epoch"`
}

func (*ArcadiaRegister) GetTypeID() uint8 {
	return ArcadiaRegisterID
}

func (a *ArcadiaRegister) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.ArcadiaRegisterKey()): state.All,
	}
}

func (*ArcadiaRegister) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16}
}

func (a *ArcadiaRegister) Execute(
	_ context.Context,
	rules chain.Rules,
	_ state.Mutable,
	_ int64,
	hght uint64,
	_ codec.Address,
	_ ids.ID,
) ([][]byte, error) {

	if a.StartEpoch < Epoch(hght, rules.GetEpochDuration())+2 {
		return nil, fmt.Errorf("epoch number is not valid, minimum: %d, actual: %d", Epoch(hght, rules.GetEpochDuration())+2, a.StartEpoch)
	}

	return nil, nil
}

func (a *ArcadiaRegister) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return ArcadiaComputeUnits
}

func (a *ArcadiaRegister) Size() int {
	return codec.BytesLen(a.Namespace) + 8
}

func (a *ArcadiaRegister) Marshal(p *codec.Packer) {
	p.PackBytes(a.Namespace)
	p.PackUint64(a.StartEpoch)
}

func (a *ArcadiaRegister) UnmarshalArcadiaRegister(p *codec.Packer) error {
	p.UnpackBytes(-1, true, &a.Namespace)
	a.StartEpoch = p.UnpackUint64(true)
	return nil
}

func (*ArcadiaRegister) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (a *ArcadiaRegister) NMTNamespace() []byte {
	return a.Namespace
}

func (*ArcadiaRegister) UseFeeMarket() bool {
	return false
}
