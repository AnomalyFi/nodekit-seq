// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
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

func (e *ImportBlockMsg) StateKeys(rauth chain.Auth, _ ids.ID) [][]byte {
	//TODO needs to be fixed
	return [][]byte{
		storage.PrefixBlockKey(e.warpMsg.StateRoot, e.warpMsg.Prnt),
	}

}

func (i *ImportBlockMsg) Execute(
	ctx context.Context,
	r chain.Rules,
	db chain.Database,
	t int64,
	rauth chain.Auth,
	_ ids.ID,
	warpVerified bool,
) (*chain.Result, error) {
	unitsUsed := i.MaxUnits(r) // max units == units
	if !warpVerified {
		return &chain.Result{
			Success: false,
			Units:   unitsUsed,
			Output:  OutputWarpVerificationFailed,
		}, nil
	}
	return &chain.Result{Success: true, Units: unitsUsed}, nil
}

func (i *ImportBlockMsg) MaxUnits(chain.Rules) uint64 {
	return uint64(len(i.warpMessage.Payload))
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

// All we encode that is action specific for now is the type byte from the
// registry.
func (i *ImportBlockMsg) Marshal(p *codec.Packer) {
	p.PackBool(i.Fill)
}

func (*ImportBlockMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
