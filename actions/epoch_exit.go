package actions

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"slices"

	hactions "github.com/AnomalyFi/hypersdk/actions"
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
	Info   hactions.EpochInfo `json:"info"`
	Epoch  uint64             `json:"epoch"`
	OpCode int                `json:"opcode"`
}

func (*EpochExit) GetTypeID() uint8 {
	return ExitID
}

func (e *EpochExit) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(hactions.EpochExitsKey(e.Epoch)):         state.All,
		string(storage.RollupInfoKey(e.Info.Namespace)): state.Read,
		string(storage.RollupRegistryKey()):             state.Read,
	}
}

func (*EpochExit) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.EpochExitsChunks, hactions.RollupInfoChunks, hactions.RollupRegistryChunks}
}

// TODO: Add check for curr epoch > start epoch of arcadia.
func (e *EpochExit) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	ts int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if e.Epoch != e.Info.Epoch {
		return nil, fmt.Errorf("epoch is not equal to what's in the meta, expected: %d, actual: %d", e.Epoch, e.Info.Epoch)
	}

	// check if rollup is registered.
	nss, err := storage.GetRollupRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}

	if !contains(nss, e.Info.Namespace) {
		return nil, fmt.Errorf("namespace is not registered, namespace: %s", hex.EncodeToString(e.Info.Namespace))
	}

	rollupInfo, err := storage.GetRollupInfo(ctx, mu, e.Info.Namespace)
	// rollup info will not be nil, as rollup is registered.
	if err != nil {
		return nil, err
	}

	if rollupInfo.AuthoritySEQAddress != actor {
		return nil, ErrNotAuthorized
	}

	epochExits, exists, err := storage.GetEpochExits(ctx, mu, e.Epoch)
	if err != nil {
		return nil, err
	}

	switch e.OpCode {
	case CreateExit:
		// Check if rollup exited already for the Epoch.
		if exists {
			for _, es := range epochExits.Exits {
				if bytes.Equal(e.Info.Namespace, es.Namespace) {
					return nil, fmt.Errorf("exit already exists, namespace: %s, epoch: %d", hex.EncodeToString(e.Info.Namespace), e.Epoch)
				}
			}
		}
		if !exists {
			epochExits = new(hactions.EpochExitInfo)
			epochExits.Exits = make([]*hactions.EpochInfo, 0)
		}
		epochExits.Exits = append(epochExits.Exits, &e.Info)
		if err := storage.SetEpochExits(ctx, mu, e.Epoch, epochExits); err != nil {
			return nil, err
		}
	case DeleteExit:
		idx := -1
		// Check if rollup exit exists and get its index in epoch exits.
		if exists {
			for i, es := range epochExits.Exits {
				if bytes.Equal(e.Info.Namespace, es.Namespace) {
					idx = i
					break
				}
			}
		}
		// Rollup did not exit prior.
		if idx == -1 {
			return nil, fmt.Errorf("exit not found, namespace: %s, epoch: %d", hex.EncodeToString(e.Info.Namespace), e.Epoch)
		}
		epochExits.Exits = slices.Delete(epochExits.Exits, idx, idx+1)
		if err := storage.SetEpochExits(ctx, mu, e.Epoch, epochExits); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("op code(%d) not supported", e.OpCode)
	}

	return nil, nil
}

func (*EpochExit) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return EpochExitComputeUnits
}

func (e *EpochExit) Size() int {
	return e.Info.Size() + consts.Uint64Len + consts.IntLen
}

func (e *EpochExit) Marshal(p *codec.Packer) {
	e.Info.Marshal(p)
	p.PackUint64(e.Epoch)
	p.PackInt(e.OpCode)
}

func UnmarshalEpochExit(p *codec.Packer) (chain.Action, error) {
	var epoch EpochExit
	info, err := hactions.UnmarshalEpochInfo(p)
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
