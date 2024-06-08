// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var _ chain.Action = (*SequencerMsg)(nil)

type SequencerMsg struct {
	ChainId     []byte        `json:"chain_id"`
	Data        []byte        `json:"data"`
	FromAddress codec.Address `json:"from_address"`
}

func (*SequencerMsg) GetTypeID() uint8 {
	return msgID
}

func (*SequencerMsg) StateKeys(_ codec.Address, actionID ids.ID) state.Keys {
	return state.Keys{
		string(storage.AssetKey(actionID)): state.Allocate | state.Write,
	}
}

// TODO fix this
func (*SequencerMsg) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.BalanceChunks}
}

func (*SequencerMsg) OutputsWarpMessage() bool {
	return false
}

func (t *SequencerMsg) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	return nil, nil
}

func (*SequencerMsg) ComputeUnits(chain.Rules) uint64 {
	return MsgComputeUnits
}

func (*SequencerMsg) Size() int {
	// TODO this should be larger because it should consider the max byte array length
	// We use size as the price of this transaction but we could just as easily
	// use any other calculation.
	return codec.AddressLen + ids.IDLen + consts.Uint64Len
}

func (t *SequencerMsg) Marshal(p *codec.Packer) {
	p.PackAddress(t.FromAddress)
	p.PackBytes(t.Data)
	p.PackBytes(t.ChainId)
}

func UnmarshalSequencerMsg(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var sequencermsg SequencerMsg
	p.UnpackAddress(false, &sequencermsg.FromAddress)
	// TODO need to correct this and check byte count
	p.UnpackBytes(-1, true, &sequencermsg.Data)
	p.UnpackBytes(-1, true, &sequencermsg.ChainId)
	return &sequencermsg, p.Err()
}

func (*SequencerMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (t *SequencerMsg) NMTNamespace() []byte {
	return t.ChainId
}
