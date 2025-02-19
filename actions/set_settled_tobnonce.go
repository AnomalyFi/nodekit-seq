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

var _ chain.Action = (*SetSettledToBNonce)(nil)

type SetSettledToBNonce struct {
	ToBNonce uint64 `json:"tobNonce"`
	Reset    bool   `json:"reset"`
}

func (*SetSettledToBNonce) GetTypeID() uint8 {
	return SetSettledToBNonceID
}

func (sst *SetSettledToBNonce) StateKeys(_ codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.DACertToBNonceKey()): state.All,
	}
}

func (*SetSettledToBNonce) StateKeysMaxChunks() []uint16 {
	return []uint16{
		storage.DACertficateToBNonceChunks, storage.DACertficateToBNonceChunks,
	}
}

func (sst *SetSettledToBNonce) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !isWhiteListed(rules, actor) {
		return nil, ErrNotWhiteListed
	}

	// store highest ToBNonce if there's new one
	tobNonce, err := storage.GetDACertToBNonce(ctx, mu)
	if err != nil {
		return nil, fmt.Errorf("failed to get tob nonce: %w", err)
	}
	if !sst.Reset && sst.ToBNonce <= tobNonce {
		return nil, fmt.Errorf("tob nonce to set is lower than in state")
	}

	// set the new tob nonce
	if err := storage.SetDACertToBNonce(ctx, mu, sst.ToBNonce); err != nil {
		return nil, fmt.Errorf("failed to store tob nonce: %w", err)
	}

	return nil, nil
}

func (*SetSettledToBNonce) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return DACertComputeUnits
}

func (sst *SetSettledToBNonce) Size() int {
	return consts.Uint64Len
}

func (sst *SetSettledToBNonce) Marshal(p *codec.Packer) {
	p.PackUint64(sst.ToBNonce)
	p.PackBool(sst.Reset)
}

func UnmarshalSetSettledToBNonce(p *codec.Packer) (chain.Action, error) {
	var sst SetSettledToBNonce
	sst.ToBNonce = p.UnpackUint64(false)
	sst.Reset = p.UnpackBool()
	return &sst, p.Err()
}

func (*SetSettledToBNonce) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*SetSettledToBNonce) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*SetSettledToBNonce) UseFeeMarket() bool {
	return false
}
