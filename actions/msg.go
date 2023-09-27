// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var _ chain.Action = (*SequencerMsg)(nil)
  
type SequencerMsg struct {
  ChainId     []byte            `protobuf:"bytes,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Data        []byte            `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	FromAddress ed25519.PublicKey `json:"from_address"`
	// `protobuf:"bytes,3,opt,name=from_address,json=fromAddress,proto3" json:"from_address,omitempty"`
}

func (*SequencerMsg) GetTypeID() uint8 {
	return msgID
}

func (t *SequencerMsg) StateKeys(rauth chain.Auth, _ ids.ID) []string {
	// owner, err := utils.ParseAddress(t.FromAddress)
	// if err != nil {
	// 	return nil, err
	// }

	return []string{
		// We always pay fees with the native asset (which is [ids.Empty])
		string(storage.BalanceKey(auth.GetActor(rauth), ids.Empty)),
		// string(t.ChainId),
		// string(t.Data),
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
	rauth chain.Auth,
	_ ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	return true, MsgComputeUnits, nil, nil, nil
}

func (*SequencerMsg) MaxComputeUnits(chain.Rules) uint64 {
	return MsgComputeUnits
}

func (*SequencerMsg) Size() int {
	//TODO this should be larger because it should consider the max byte array length
	// We use size as the price of this transaction but we could just as easily
	// use any other calculation.
	return ed25519.PublicKeyLen + consts.IDLen + consts.Uint64Len
}

func (t *SequencerMsg) Marshal(p *codec.Packer) {
	p.PackPublicKey(t.FromAddress)
	p.PackBytes(t.Data)
	p.PackBytes(t.ChainId)
}

func UnmarshalSequencerMsg(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var sequencermsg SequencerMsg
	p.UnpackPublicKey(false, &sequencermsg.FromAddress)
	// TODO need to correct this and check byte count
	p.UnpackBytes(-1, true, &sequencermsg.Data)
	p.UnpackBytes(-1, true, &sequencermsg.ChainId)
	return &sequencermsg, p.Err()
}

func (*SequencerMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
