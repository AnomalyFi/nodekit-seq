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

var _ chain.Action = (*ImportMsg)(nil)

type ImportMsg struct {
	// warpTransfer is parsed from the inner *warp.Message
	warpMsg *WarpSequencerMsg

	// warpMessage is the full *warp.Message parsed from [chain.Transaction]
	warpMessage *warp.Message
}

func (i *ImportMsg) StateKeys(rauth chain.Auth, _ ids.ID) [][]byte {
	//TODO this needs to be fixed
	var (
		keys    [][]byte
		assetID ids.ID
	)
	assetID = ImportedMsgID(i.warpMessage.SourceChainID)
	keys = [][]byte{
		storage.PrefixAssetKey(assetID),
		storage.PrefixBalanceKey(i.warpMsg.FromAddress, assetID),
	}

	return keys
}

func (i *ImportMsg) Execute(
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

func (i *ImportMsg) MaxUnits(chain.Rules) uint64 {
	return uint64(len(i.warpMessage.Payload))
}

func UnmarshalImportMsg(p *codec.Packer, wm *warp.Message) (chain.Action, error) {
	var (
		imp ImportMsg
		err error
	)
	if err := p.Err(); err != nil {
		return nil, err
	}
	imp.warpMessage = wm
	imp.warpMsg, err = UnmarshalWarpSequencerMsg(imp.warpMessage.Payload)
	if err != nil {
		return nil, err
	}

	return &imp, nil
}

func (*ImportMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
