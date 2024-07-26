// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*SequencerMsg)(nil)

type SequencerMsg struct {
	ChainId     []byte        `json:"chain_id"`
	Data        []byte        `json:"data"`
	FromAddress codec.Address `json:"from_address"`
	RelayerID   int           `json:"relayer_id"`
}

func (*SequencerMsg) GetTypeID() uint8 {
	return msgID
}

func (*SequencerMsg) StateKeys(_ codec.Address, actionID ids.ID) state.Keys {
	return state.Keys{}
}

// TODO fix this
func (*SequencerMsg) StateKeysMaxChunks() []uint16 {
	return []uint16{}
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

func (msg *SequencerMsg) Size() int {
	return codec.BytesLen(msg.ChainId) + codec.BytesLen(msg.Data) + codec.AddressLen + consts.IntLen
}

func (t *SequencerMsg) Marshal(p *codec.Packer) {
	p.PackAddress(t.FromAddress)
	p.PackBytes(t.Data)
	p.PackBytes(t.ChainId)
	p.PackInt(t.RelayerID)
}

func UnmarshalSequencerMsg(p *codec.Packer) (chain.Action, error) {
	var sequencermsg SequencerMsg
	p.UnpackAddress(&sequencermsg.FromAddress)
	// TODO need to correct this and check byte count
	p.UnpackBytes(-1, true, &sequencermsg.Data)
	p.UnpackBytes(-1, true, &sequencermsg.ChainId)
	// Note, required has to be false or RelayerID of 0 will report ID not populated
	sequencermsg.RelayerID = p.UnpackInt(false)
	return &sequencermsg, p.Err()
}

func (*SequencerMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (t *SequencerMsg) NMTNamespace() []byte {
	return t.ChainId
}
