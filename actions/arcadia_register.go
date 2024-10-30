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

var _ chain.Action = (*ArcadiaRegistration)(nil)

const (
	CreateArcadia = iota
	DeleteArcadia
	UpdateArcadia
)

type ArcadiaRegistration struct {
	Info       hactions.RollupInfo `json:"info"`
	Namespace  []byte              `json:"namespace"`
	StartEpoch uint64              `json:"startEpoch"`
	OpCode     int                 `json:"opcode"`
}

func (*ArcadiaRegistration) GetTypeID() uint8 {
	return hactions.ArcadiaRegisterID
}

func (a *ArcadiaRegistration) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.RollupInfoKey(a.Namespace)): state.All,
		string(storage.ArcadiaRegistryKey()):       state.All,
		string(storage.GlobalRollupRegistryKey()):  state.All,
	}
}

func (*ArcadiaRegistration) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.RollupInfoChunks, hactions.ArcadiaRegistryChunks, storage.GlobalRollupRegistryChunks}
}

func (a *ArcadiaRegistration) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	hght uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !bytes.Equal(a.Namespace, a.Info.Namespace) {
		return nil, fmt.Errorf("namespace is not equal to what's in the meta, expected: %s, actual: %s", hex.EncodeToString(a.Namespace), hex.EncodeToString(a.Info.Namespace))
	}

	if len(a.Namespace) > consts.MaxNamespaceLen {
		return nil, fmt.Errorf("namespace length is too long, maximum: %d, actual: %d", consts.MaxNamespaceLen, len(a.Namespace))
	}
	if a.StartEpoch < Epoch(hght, rules.GetEpochDuration())+2 { // @todo derive this const from hypersdk preconf level.
		return nil, fmt.Errorf("epoch number is not valid, minimum: %d, actual: %d", Epoch(hght, rules.GetEpochDuration())+2, a.StartEpoch)
	}
	namespaces, err := storage.GetArcadiaRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}
	globalRollupNamespaces, err := storage.GetGlobalRollupRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}

	switch a.OpCode {
	case CreateArcadia:
		if contains(globalRollupNamespaces, a.Namespace) {
			return nil, ErrNameSpaceAlreadyRegistered
		}
		namespaces = append(namespaces, a.Namespace)
		globalRollupNamespaces = append(globalRollupNamespaces, a.Namespace)
		if err := storage.SetRollupInfo(ctx, mu, a.Namespace, &a.Info); err != nil {
			return nil, err
		}
	case UpdateArcadia:
		if err := authorizationChecks(ctx, actor, namespaces, a.Namespace, mu); err != nil {
			return nil, err
		}

		if err := storage.SetRollupInfo(ctx, mu, a.Namespace, &a.Info); err != nil {
			return nil, err
		}
	case DeleteArcadia:
		if err := authorizationChecks(ctx, actor, namespaces, a.Namespace, mu); err != nil {
			return nil, err
		}

		nsIdx := -1
		for i, ns := range namespaces {
			if bytes.Equal(a.Namespace, ns) {
				nsIdx = i
				break
			}
		}
		namespaces = slices.Delete(namespaces, nsIdx, nsIdx+1)

		nsIdx = -1
		for i, ns := range globalRollupNamespaces {
			if bytes.Equal(a.Namespace, ns) {
				nsIdx = i
				break
			}
		}
		globalRollupNamespaces = slices.Delete(globalRollupNamespaces, nsIdx, nsIdx+1)

		if err := storage.DelRollupInfo(ctx, mu, a.Namespace); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("op code(%d) not supported", a.OpCode)
	}
	if err := storage.SetArcadiaRegistry(ctx, mu, namespaces); err != nil {
		return nil, err
	}
	if err := storage.SetGlobalRollupRegistry(ctx, mu, globalRollupNamespaces); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *ArcadiaRegistration) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return hactions.ArcadiaRegisterComputeUnits
}

func (a *ArcadiaRegistration) Size() int {
	return a.Info.Size() + codec.BytesLen(a.Namespace) + consts.Uint64Len + consts.IntLen
}

func (a *ArcadiaRegistration) Marshal(p *codec.Packer) {
	a.Info.Marshal(p)
	p.PackBytes(a.Namespace)
	p.PackUint64(a.StartEpoch)
	p.PackInt(a.OpCode)
}

func UnmarshalArcadiaRegister(p *codec.Packer) (chain.Action, error) {
	var arcadiaReg ArcadiaRegistration
	info, err := hactions.UnmarshalRollupInfo(p)
	if err != nil {
		return nil, err
	}
	arcadiaReg.Info = *info
	p.UnpackBytes(-1, true, &arcadiaReg.Namespace)
	arcadiaReg.StartEpoch = p.UnpackUint64(false)
	arcadiaReg.OpCode = p.UnpackInt(true)
	return &arcadiaReg, nil
}

func (*ArcadiaRegistration) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (a *ArcadiaRegistration) NMTNamespace() []byte {
	return a.Namespace
}

func (*ArcadiaRegistration) UseFeeMarket() bool {
	return false
}
