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

var _ chain.Action = (*RollupRegistration)(nil)

const (
	CreateRollup = iota
	DeleteRollup
	UpdateRollup
)

type RollupRegistration struct {
	Info       hactions.RollupInfo `json:"info"`
	Namespace  []byte              `json:"namespace"`
	StartEpoch uint64              `json:"startEpoch"`
	OpCode     int                 `json:"opcode"`
}

func (*RollupRegistration) GetTypeID() uint8 {
	return hactions.RollupRegisterID
}

func (r *RollupRegistration) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.RollupInfoKey(r.Namespace)): state.All,
		string(storage.RollupRegistryKey()):        state.All,
	}
}

func (*RollupRegistration) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.RollupInfoChunks, hactions.RollupRegistryChunks}
}

func (r *RollupRegistration) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	hght uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !bytes.Equal(r.Namespace, r.Info.Namespace) {
		return nil, fmt.Errorf("namespace is not equal to what's in the meta, expected: %s, actual: %s", hex.EncodeToString(r.Namespace), hex.EncodeToString(r.Info.Namespace))
	}

	if len(r.Namespace) > consts.MaxNamespaceLen {
		return nil, fmt.Errorf("namespace length is too long, maximum: %d, actual: %d", consts.MaxNamespaceLen, len(r.Namespace))
	}

	namespaces, err := storage.GetRollupRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}

	switch r.OpCode {
	case CreateRollup:
		if contains(namespaces, r.Namespace) {
			return nil, ErrNameSpaceAlreadyRegistered
		}
		if r.StartEpoch < Epoch(hght, rules.GetEpochLength())+2 {
			return nil, fmt.Errorf("epoch number is not valid, minimum: %d, actual: %d", Epoch(hght, rules.GetEpochLength())+2, r.StartEpoch)
		}
		namespaces = append(namespaces, r.Namespace)
		if err := storage.SetRollupInfo(ctx, mu, r.Namespace, &r.Info); err != nil {
			return nil, err
		}
	case UpdateRollup:
		if err := authorizationChecks(ctx, actor, namespaces, r.Namespace, mu); err != nil {
			return nil, err
		}

		if err := storage.SetRollupInfo(ctx, mu, r.Namespace, &r.Info); err != nil {
			return nil, err
		}
	case DeleteRollup:
		if err := authorizationChecks(ctx, actor, namespaces, r.Namespace, mu); err != nil {
			return nil, err
		}

		nsIdx := -1
		for i, ns := range namespaces {
			if bytes.Equal(r.Namespace, ns) {
				nsIdx = i
				break
			}
		}
		namespaces = slices.Delete(namespaces, nsIdx, nsIdx+1)

		if err := storage.DelRollupInfo(ctx, mu, r.Namespace); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("op code(%d) not supported", r.OpCode)
	}
	if err := storage.SetRollupRegistry(ctx, mu, namespaces); err != nil {
		return nil, err
	}

	return nil, nil
}

func (*RollupRegistration) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return hactions.RollupRegisterComputeUnits
}

func (r *RollupRegistration) Size() int {
	return r.Info.Size() + len(r.Namespace) + consts.Uint64Len + consts.IntLen
}

func (r *RollupRegistration) Marshal(p *codec.Packer) {
	r.Info.Marshal(p)
	p.PackBytes(r.Namespace)
	p.PackInt(r.OpCode)
	p.PackUint64(r.StartEpoch)
}

func UnmarshalRollupRegister(p *codec.Packer) (chain.Action, error) {
	var RollupReg RollupRegistration
	info, err := hactions.UnmarshalRollupInfo(p)
	if err != nil {
		return nil, err
	}
	RollupReg.Info = *info
	p.UnpackBytes(-1, false, &RollupReg.Namespace)
	RollupReg.OpCode = p.UnpackInt(false)
	if RollupReg.OpCode == CreateRollup {
		RollupReg.StartEpoch = p.UnpackUint64(true)
	} else {
		RollupReg.StartEpoch = p.UnpackUint64(false)
	}
	return &RollupReg, nil
}

func (*RollupRegistration) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (*RollupRegistration) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*RollupRegistration) UseFeeMarket() bool {
	return false
}
