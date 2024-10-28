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

var _ chain.Action = (*AnchorRegister)(nil)

const (
	CreateAnchor = iota
	DeleteAnchor
	UpdateAnchor
)

type AnchorRegister struct {
	Info      hactions.AnchorInfo `json:"info"`
	Namespace []byte              `json:"namespace"`
	OpCode    int                 `json:"opcode"`
}

func (*AnchorRegister) GetTypeID() uint8 {
	return hactions.AnchorRegisterID
}

func (t *AnchorRegister) StateKeys(actor codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.AnchorKey(t.Namespace)): state.All,
		string(storage.AnchorRegistryKey()):    state.All,
	}
}

func (*AnchorRegister) StateKeysMaxChunks() []uint16 {
	return []uint16{hactions.AnchorChunks, hactions.AnchorChunks}
}

func (t *AnchorRegister) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !bytes.Equal(t.Namespace, t.Info.Namespace) {
		return nil, fmt.Errorf("namespace is not equal to what's in the meta, expected: %s, actual: %s", hex.EncodeToString(t.Namespace), hex.EncodeToString(t.Info.Namespace))
	}

	namespaces, _, err := storage.GetAnchors(ctx, mu)
	if err != nil {
		return nil, err
	}

	switch t.OpCode {
	case CreateAnchor:
		namespaces = append(namespaces, t.Namespace)
		if err := storage.SetAnchor(ctx, mu, t.Namespace, &t.Info); err != nil {
			return nil, err
		}
	case UpdateAnchor:
		namespaces = append(namespaces, t.Namespace)
		if err := storage.SetAnchor(ctx, mu, t.Namespace, &t.Info); err != nil {
			return nil, err
		}
	case DeleteAnchor:
		nsIdx := -1
		for i, ns := range namespaces {
			if bytes.Equal(t.Namespace, ns) {
				nsIdx = i
				break
			}
		}
		namespaces = slices.Delete(namespaces, nsIdx, nsIdx+1)
		if err := storage.DelAnchor(ctx, mu, t.Namespace); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("op code(%d) not supported", t.OpCode)
	}

	if err := storage.SetAnchors(ctx, mu, namespaces); err != nil {
		return nil, err
	}

	return nil, nil
}

func (*AnchorRegister) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return hactions.AnchorRegisterComputeUnits
}

func (t *AnchorRegister) Size() int {
	return codec.BytesLen(t.Namespace) + codec.AddressLen + consts.BoolLen
}

func (t *AnchorRegister) Marshal(p *codec.Packer) {
	t.Info.Marshal(p)
	p.PackBytes(t.Namespace)
	p.PackInt(t.OpCode)
}

func UnmarshalAnchorRegister(p *codec.Packer) (chain.Action, error) {
	var anchorReg AnchorRegister
	info, err := hactions.UnmarshalAnchorInfo(p)
	if err != nil {
		return nil, err
	}
	anchorReg.Info = *info
	p.UnpackBytes(-1, false, &anchorReg.Namespace)
	anchorReg.OpCode = p.UnpackInt(false)
	return &anchorReg, nil
}

func (*AnchorRegister) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*AnchorRegister) NMTNamespace() []byte {
	return DefaultNMTNamespace // TODO: mark this the same to registering namespace?
}

func (*AnchorRegister) UseFeeMarket() bool {
	return false
}
