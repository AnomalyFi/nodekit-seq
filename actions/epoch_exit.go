package actions

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*EpochExit)(nil)

const (
	CreateExit = iota
	DeleteExit
)

type EpochExit struct {
	Info   storage.EpochInfo `json:"info"`
	Epoch  uint64            `json:"epoch"`
	OpCode int               `json:"opcode"`
}

func (*EpochExit) GetTypeID() uint8 {
	return ExitID
}

func (t *EpochExit) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.EpochExitKey(t.Epoch)):  state.All,
		string(storage.EpochExitRegistryKey()): state.All,
	}
}

func (*EpochExit) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.EpochExitChunks, storage.EpochExitChunks}
}

func (*EpochExit) OutputsWarpMessage() bool {
	return false
}

func (t *EpochExit) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if t.Epoch != t.Info.Epoch {
		return nil, fmt.Errorf("Epoch is not equal to what's in the meta, expected: %d, actual: %d", t.Epoch, t.Info.Epoch)
	}

	epochExit, err := storage.GetEpochExit(ctx, mu, t.Epoch)

	// namespaces, _, err := storage.GetAnchors(ctx, mu)
	if err != nil {
		return nil, err
	}

	switch t.OpCode {
	case CreateExit:
		epochExit.Exits = append(epochExit.Exits, t.Info)
		if err := storage.SetEpochExit(ctx, mu, t.Epoch, epochExit); err != nil {
			return nil, err
		}
	case DeleteExit:
		idx := -1
		for i, e := range epochExit.Exits {
			if t.Epoch == e.Epoch && bytes.Equal(t.Info.Namespace, e.Namespace) {
				idx = i
				break
			}
		}
		epochExit.Exits = slices.Delete(epochExit.Exits, idx, idx+1)
		if err := storage.SetEpochExit(ctx, mu, t.Epoch, epochExit); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("op code(%d) not supported", t.OpCode)
	}

	// if err := storage.SetAnchors(ctx, mu, namespaces); err != nil {
	// 	return nil, err
	// }

	return nil, nil
}

func (*EpochExit) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return EpochExitComputeUnits
}

func (t *EpochExit) Size() int {
	return codec.BytesLen(t.Info.Namespace) + consts.Uint64Len + consts.BoolLen
}

func (t *EpochExit) Marshal(p *codec.Packer) {
	t.Info.Marshal(p)
	p.PackUint64(t.Epoch)
	p.PackInt(t.OpCode)
}

func UnmarshalEpochExit(p *codec.Packer) (chain.Action, error) {
	var epoch EpochExit
	info, err := storage.UnmarshalEpochInfo(p)
	if err != nil {
		return nil, err
	}
	epoch.Info = *info
	epoch.Epoch = p.UnpackUint64(true)
	epoch.OpCode = p.UnpackInt(false)
	return &epoch, nil
}

func (*EpochExit) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*EpochExit) NMTNamespace() []byte {
	return DefaultNMTNamespace // TODO: mark this the same to registering namespace?
}

func (*EpochExit) UseFeeMarket() bool {
	return false
}
