package actions

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

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
	ExitRollup
	UpdateRollup
)

type RollupRegistration struct {
	Info      hactions.RollupInfo `json:"info"`
	Namespace []byte              `json:"namespace"`
	OpCode    int                 `json:"opcode"`
}

func (*RollupRegistration) GetTypeID() uint8 {
	return hactions.RollupRegisterID
}

func (r *RollupRegistration) StateKeys(_ codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.RollupInfoKey(r.Namespace)): state.All,
		string(storage.RollupRegistryKey()):        state.All,
	}
}

func (*RollupRegistration) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.RollupInfoChunks, hactions.RollupRegistryChunks}
}

// TODO: this action needs to be managed by DAO to manage deletions since we are not deleting any namespace from storage
// but only by marking them as regsitered or exited
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
		return nil, fmt.Errorf("unable to get namespaces: %w", err)
	}

	switch r.OpCode {
	case CreateRollup:
		if contains(namespaces, r.Namespace) {
			return nil, ErrNameSpaceAlreadyRegistered
		}
		if r.Info.StartEpoch < Epoch(hght, rules.GetEpochLength())+2 || r.Info.ExitEpoch != 0 {
			return nil, fmt.Errorf("epoch number is not valid, minimum: %d, actual: %d, exit: %d", Epoch(hght, rules.GetEpochLength())+2, r.Info.StartEpoch, r.Info.ExitEpoch)
		}
		namespaces = append(namespaces, r.Namespace)
		if err := storage.SetRollupInfo(ctx, mu, r.Namespace, &r.Info); err != nil {
			return nil, fmt.Errorf("unable to set rollup info(CREATE): %w", err)
		}
	case UpdateRollup:
		// only allow modifing informations that are not related to ExitEpoch or StartEpoch
		if err := authorizationChecks(ctx, actor, namespaces, r.Namespace, mu); err != nil {
			return nil, fmt.Errorf("authorization failed(UPDATE): %w", err)
		}

		rollupInfoExists, err := storage.GetRollupInfo(ctx, mu, r.Namespace)
		if err != nil {
			return nil, fmt.Errorf("unable to get existing rollup info(UPDATE): %w", err)
		}

		// rewrite epoch info
		r.Info.ExitEpoch = rollupInfoExists.ExitEpoch
		r.Info.StartEpoch = rollupInfoExists.StartEpoch

		if err := storage.SetRollupInfo(ctx, mu, r.Namespace, &r.Info); err != nil {
			return nil, fmt.Errorf("unable to set rollup info(UPDATE): %w", err)
		}
	case ExitRollup:
		if err := authorizationChecks(ctx, actor, namespaces, r.Namespace, mu); err != nil {
			return nil, fmt.Errorf("unable to set rollup info(EXIT): %w", err)
		}
		rollupInfoExists, err := storage.GetRollupInfo(ctx, mu, r.Namespace)
		if err != nil {
			return nil, fmt.Errorf("unable to get existing rollup info(UPDATE): %w", err)
		}
		if r.Info.ExitEpoch < rollupInfoExists.StartEpoch || r.Info.ExitEpoch < Epoch(hght, rules.GetEpochLength())+2 {
			return nil, fmt.Errorf("(EXIT)epoch number is not valid, minimum: %d, actual: %d, start: %d", Epoch(hght, rules.GetEpochLength())+2, r.Info.ExitEpoch, rollupInfoExists.StartEpoch)
		}

		// overwrite StartEpoch
		r.Info.StartEpoch = rollupInfoExists.StartEpoch

		if err := storage.SetRollupInfo(ctx, mu, r.Namespace, &r.Info); err != nil {
			return nil, fmt.Errorf("unable to set rollup info(EXIT): %w", err)
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
}

func UnmarshalRollupRegister(p *codec.Packer) (chain.Action, error) {
	var rollupReg RollupRegistration
	info, err := hactions.UnmarshalRollupInfo(p)
	if err != nil {
		return nil, err
	}
	rollupReg.Info = *info
	p.UnpackBytes(-1, false, &rollupReg.Namespace)
	rollupReg.OpCode = p.UnpackInt(false)
	return &rollupReg, nil
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
