// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var _ chain.Action = (*CreateAsset)(nil)

type CreateAsset struct {
	Symbol   []byte `json:"symbol"`
	Decimals uint8  `json:"decimals"`
	Metadata []byte `json:"metadata"`
}

func (*CreateAsset) GetTypeID() uint8 {
	return createAssetID
}

func (*CreateAsset) StateKeys(_ codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.AssetKey(txID)): state.Allocate | state.Write,
	}
}

func (*CreateAsset) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.AssetChunks}
}

func (*CreateAsset) OutputsWarpMessage() bool {
	return false
}

func (c *CreateAsset) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if len(c.Symbol) == 0 {
		return false, CreateAssetComputeUnits, OutputSymbolEmpty, nil, nil
	}
	if len(c.Symbol) > MaxSymbolSize {
		return false, CreateAssetComputeUnits, OutputSymbolTooLarge, nil, nil
	}
	if c.Decimals > MaxDecimals {
		return false, CreateAssetComputeUnits, OutputDecimalsTooLarge, nil, nil
	}
	if len(c.Metadata) == 0 {
		return false, CreateAssetComputeUnits, OutputMetadataEmpty, nil, nil
	}
	if len(c.Metadata) > MaxMetadataSize {
		return false, CreateAssetComputeUnits, OutputMetadataTooLarge, nil, nil
	}
	// It should only be possible to overwrite an existing asset if there is
	// a hash collision.
	if err := storage.SetAsset(ctx, mu, txID, c.Symbol, c.Decimals, c.Metadata, 0, actor, false); err != nil {
		return false, CreateAssetComputeUnits, utils.ErrBytes(err), nil, nil
	}
	return true, CreateAssetComputeUnits, nil, nil, nil
}

func (*CreateAsset) MaxComputeUnits(chain.Rules) uint64 {
	return CreateAssetComputeUnits
}

func (c *CreateAsset) Size() int {
	// TODO: add small bytes (smaller int prefix)
	return codec.BytesLen(c.Symbol) + consts.Uint8Len + codec.BytesLen(c.Metadata)
}

func (c *CreateAsset) Marshal(p *codec.Packer) {
	p.PackBytes(c.Symbol)
	p.PackByte(c.Decimals)
	p.PackBytes(c.Metadata)
}

func UnmarshalCreateAsset(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var create CreateAsset
	p.UnpackBytes(MaxSymbolSize, true, &create.Symbol)
	create.Decimals = p.UnpackByte()
	p.UnpackBytes(MaxMetadataSize, true, &create.Metadata)
	return &create, p.Err()
}

func (*CreateAsset) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*CreateAsset) NMTNamespace() []byte {
	return DefaultNMTNamespace
}
