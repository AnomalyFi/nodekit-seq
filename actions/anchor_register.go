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

var _ chain.Action = (*AnchorRegistration)(nil)

const (
	CreateAnchor = iota
	DeleteAnchor
	UpdateAnchor
)

type AnchorRegistration struct {
	Info      hactions.RollupInfo `json:"info"`
	Namespace []byte              `json:"namespace"`
	OpCode    int                 `json:"opcode"`
}

func (*AnchorRegistration) GetTypeID() uint8 {
	return hactions.AnchorRegisterID
}

func (a *AnchorRegistration) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.RollupInfoKey(a.Namespace)): state.All,
		string(storage.AnchorRegistryKey()):        state.All,
		string(storage.GlobalRollupRegistryKey()):  state.All,
	}
}

func (*AnchorRegistration) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.RollupInfoChunks, hactions.AnchorRegistryChunks, storage.GlobalRollupRegistryChunks}
}

func (a *AnchorRegistration) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !bytes.Equal(a.Namespace, a.Info.Namespace) {
		return nil, fmt.Errorf("namespace is not equal to what's in the meta, expected: %s, actual: %s", hex.EncodeToString(a.Namespace), hex.EncodeToString(a.Info.Namespace))
	}

	if len(a.Namespace) > consts.MaxNamespaceLen {
		return nil, fmt.Errorf("namespace length is too long, maximum: %d, actual: %d", consts.MaxNamespaceLen, len(a.Namespace))
	}

	namespaces, err := storage.GetAnchorRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}

	globalRollupNamespaces, err := storage.GetGlobalRollupRegistry(ctx, mu)
	if err != nil {
		return nil, err
	}

	switch a.OpCode {
	case CreateAnchor:
		if contains(globalRollupNamespaces, a.Namespace) {
			return nil, ErrNameSpaceAlreadyRegistered
		}
		namespaces = append(namespaces, a.Namespace)
		globalRollupNamespaces = append(globalRollupNamespaces, a.Namespace)
		if err := storage.SetRollupInfo(ctx, mu, a.Namespace, &a.Info); err != nil {
			return nil, err
		}
	case UpdateAnchor:
		if err := authorizationChecks(ctx, actor, namespaces, a.Namespace, mu); err != nil {
			return nil, err
		}

		if err := storage.SetRollupInfo(ctx, mu, a.Namespace, &a.Info); err != nil {
			return nil, err
		}
	case DeleteAnchor:
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
	if err := storage.SetAnchorRegistry(ctx, mu, namespaces); err != nil {
		return nil, err
	}
	if err := storage.SetGlobalRollupRegistry(ctx, mu, globalRollupNamespaces); err != nil {
		return nil, err
	}

	return nil, nil
}

func (*AnchorRegistration) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return hactions.AnchorRegisterComputeUnits
}

func (a *AnchorRegistration) Size() int {
	return a.Info.Size() + len(a.Namespace) + consts.IntLen
}

func (a *AnchorRegistration) Marshal(p *codec.Packer) {
	a.Info.Marshal(p)
	p.PackBytes(a.Namespace)
	p.PackInt(a.OpCode)
}

func UnmarshalAnchorRegister(p *codec.Packer) (chain.Action, error) {
	var anchorReg AnchorRegistration
	info, err := hactions.UnmarshalRollupInfo(p)
	if err != nil {
		return nil, err
	}
	anchorReg.Info = *info
	p.UnpackBytes(-1, false, &anchorReg.Namespace)
	anchorReg.OpCode = p.UnpackInt(false)
	return &anchorReg, nil
}

func (*AnchorRegistration) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (*AnchorRegistration) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*AnchorRegistration) UseFeeMarket() bool {
	return false
}
