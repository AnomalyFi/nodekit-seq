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
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/math"
)

var _ chain.Action = (*SequencerMsg)(nil)

type SequencerMsg struct {
	ChainId     []byte        `json:"chain_id"`
	Data        []byte        `json:"data"`
	FromAddress codec.Address `json:"from_address"`
	RelayerID   uint32        `json:"relayer_id"`
}

func (*SequencerMsg) GetTypeID() uint8 {
	return msgID
}

func (s *SequencerMsg) StateKeys(actor codec.Address, actionID ids.ID) state.Keys {
	return state.Keys{
		string(storage.RelayerGasPriceKey(s.RelayerID)): state.Read,
		string(storage.BalanceKey(actor, ids.Empty)):    state.All,
		string(storage.RelayerBalanceKey(s.RelayerID)):  state.All,
	}
}

func (*SequencerMsg) StateKeysMaxChunks() []uint16 {
	return []uint16{
		storage.RelayerGasChunks,
		storage.BalanceChunks,
		storage.RelayerGasChunks,
	}
}

func (s *SequencerMsg) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	price, err := storage.GetRelayerGasPrice(ctx, mu, s.RelayerID)
	if err != nil && err != database.ErrNotFound {
		return nil, err
	}
	cost, err := math.Mul64(price, uint64(len(s.Data)))
	if err != nil {
		return nil, err
	}
	// deduct DA costs from the actor's balance
	if err := storage.SubBalance(ctx, mu, actor, ids.Empty, cost); err != nil {
		return nil, err
	}
	// add DA costs to the relayer's balance
	if err := storage.AddRelayerBalance(ctx, mu, s.RelayerID, cost); err != nil {
		return nil, err
	}
	return nil, nil
}

func (*SequencerMsg) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return MsgComputeUnits
}

func (s *SequencerMsg) Size() int {
	return codec.AddressLen + consts.Uint64Len + len(s.ChainId) + len(s.Data)
}

func (s *SequencerMsg) Marshal(p *codec.Packer) {
	p.PackAddress(s.FromAddress)
	p.PackBytes(s.Data)
	p.PackBytes(s.ChainId)
	p.PackUint64(uint64(s.RelayerID))
}

func UnmarshalSequencerMsg(p *codec.Packer) (chain.Action, error) {
	var sequencermsg SequencerMsg
	p.UnpackAddress(&sequencermsg.FromAddress)
	// TODO need to correct this and check byte count
	p.UnpackBytes(-1, true, &sequencermsg.Data)
	p.UnpackBytes(-1, true, &sequencermsg.ChainId)
	sequencermsg.RelayerID = uint32(p.UnpackUint64(true))
	return &sequencermsg, p.Err()
}

func (*SequencerMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (s *SequencerMsg) NMTNamespace() []byte {
	return s.ChainId
}

func (*SequencerMsg) UseFeeMarket() bool {
	return true
}
