// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
)

var _ chain.Action = (*ImportBlockMsg)(nil)

type ImportBlockMsg struct {
	// Fill indicates if the actor wishes to fill the order request in the warp
	// message. This must be true if the warp message is in a block with
	// a timestamp < [SwapExpiry].
	Fill bool `json:"fill"`
	// warpTransfer is parsed from the inner *warp.Message
	warpMsg *chain.WarpBlock

	// warpMessage is the full *warp.Message parsed from [chain.Transaction]
	warpMessage *warp.Message
}

func (e *ImportBlockMsg) StateKeys(rauth chain.Auth, _ ids.ID) []string {
	// TODO needs to be fixed
	return []string{
		string(storage.PrefixBlockKey(e.warpMsg.StateRoot, e.warpMsg.Prnt)),
	}
}

func (i *ImportBlockMsg) StateKeysMaxChunks() []uint16 {
	// TODO need to fix this
	// Can't use [warpTransfer] because it may not be populated yet
	chunks := []uint16{}
	chunks = append(chunks, storage.LoanChunks)
	chunks = append(chunks, storage.AssetChunks)
	chunks = append(chunks, storage.BalanceChunks)

	// If the [warpTransfer] specified a reward, we add the state key to make
	// sure it is paid.
	chunks = append(chunks, storage.BalanceChunks)

	// If the [warpTransfer] requests a swap, we add the state keys to transfer
	// the required balances.
	if i.Fill {
		chunks = append(chunks, storage.BalanceChunks)
		chunks = append(chunks, storage.BalanceChunks)
		chunks = append(chunks, storage.BalanceChunks)
	}
	return chunks
}

func (i *ImportBlockMsg) Execute(
	ctx context.Context,
	r chain.Rules,
	mu state.Mutable,
	t int64,
	rauth chain.Auth,
	_ ids.ID,
	warpVerified bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if !warpVerified {
		return false, ImportBlockComputeUnits, OutputValueZero, nil, nil
	}
	return true, ImportBlockComputeUnits, nil, nil, nil
}

func (i *ImportBlockMsg) MaxComputeUnits(chain.Rules) uint64 {
	return ImportBlockComputeUnits
}

func (*ImportBlockMsg) Size() int {
	return consts.BoolLen
}

func (*ImportBlockMsg) OutputsWarpMessage() bool {
	return false
}

func (*ImportBlockMsg) GetTypeID() uint8 {
	return importBlockID
}

// All we encode that is action specific for now is the type byte from the
// registry.
func (i *ImportBlockMsg) Marshal(p *codec.Packer) {
	p.PackBool(i.Fill)
}

func UnmarshalImportBlockMsg(p *codec.Packer, wm *warp.Message) (chain.Action, error) {
	var (
		imp ImportBlockMsg
		err error
	)
	imp.Fill = p.UnpackBool()
	if err := p.Err(); err != nil {
		return nil, err
	}
	imp.warpMessage = wm
	imp.warpMsg, err = chain.UnmarshalWarpBlock(imp.warpMessage.Payload)
	if err != nil {
		return nil, err
	}

	return &imp, nil
}

func (*ImportBlockMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*ImportBlockMsg) NMTNamespace() []byte {
	return DefaultNMTNamespace
}
