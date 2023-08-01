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
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/storage"
)

var _ chain.Action = (*ExportBlockMsg)(nil)

type ExportBlockMsg struct {
	Prnt        ids.ID `json:"parent"`
	Tmstmp      int64  `json:"timestamp"`
	Hght        uint64 `json:"height"`
	StateRoot   ids.ID `json:"stateRoot"`
	Destination ids.ID `json:"destination"`
}

func (e *ExportBlockMsg) StateKeys(rauth chain.Auth, _ ids.ID) [][]byte {
	//TODO needs to be fixed
	// var (
	// 	keys  [][]byte
	// 	actor = auth.GetActor(rauth)
	// )
	// if e.Return {
	// 	keys = [][]byte{
	// 		storage.PrefixAssetKey(e.Asset),
	// 		storage.PrefixBalanceKey(actor, e.Asset),
	// 	}
	// } else {
	// 	keys = [][]byte{
	// 		storage.PrefixAssetKey(e.Asset),
	// 		storage.PrefixLoanKey(e.Asset, e.Destination),
	// 		storage.PrefixBalanceKey(actor, e.Asset),
	// 	}
	// }

	return [][]byte{
		storage.PrefixBlockKey(e.StateRoot, e.Destination),
		storage.PrefixBlockKey(e.Prnt, e.Destination),
	}
}

func (e *ExportBlockMsg) executeLoan(
	ctx context.Context,
	r chain.Rules,
	db chain.Database,
	actor crypto.PublicKey,
	txID ids.ID,
) (*chain.Result, error) {
	unitsUsed := e.MaxUnits(r)

	wt := &chain.WarpBlock{
		Prnt:      e.Prnt,
		Tmstmp:    e.Tmstmp,
		Hght:      e.Hght,
		StateRoot: e.StateRoot,
		TxID:      txID,
	}
	payload, err := wt.Marshal()
	if err != nil {
		return &chain.Result{Success: false, Units: unitsUsed, Output: utils.ErrBytes(err)}, nil
	}
	wm := &warp.UnsignedMessage{
		DestinationChainID: e.Destination,
		// SourceChainID is populated by hypersdk
		Payload: payload,
	}
	return &chain.Result{Success: true, Units: unitsUsed, WarpMessage: wm}, nil
}

func (e *ExportBlockMsg) Execute(
	ctx context.Context,
	r chain.Rules,
	db chain.Database,
	_ int64,
	rauth chain.Auth,
	txID ids.ID,
	_ bool,
) (*chain.Result, error) {

	actor := auth.GetActor(rauth)
	unitsUsed := e.MaxUnits(r) // max units == units
	if e.Destination == ids.Empty {
		// This would result in multiplying balance export by whoever imports the
		// transaction.
		return &chain.Result{Success: false, Units: unitsUsed, Output: OutputAnycast}, nil
	}
	// TODO: check if destination is ourselves
	// if e.Return {
	// 	return e.executeReturn(ctx, r, db, actor, txID)
	// }
	return e.executeLoan(ctx, r, db, actor, txID)
}

func (*ExportBlockMsg) MaxUnits(chain.Rules) uint64 {
	//TODO fix this
	return consts.IDLen +
		consts.Uint64Len + consts.Uint64Len +
		consts.IDLen + consts.IDLen
}

func (e *ExportBlockMsg) Marshal(p *codec.Packer) {
	p.PackID(e.Prnt)
	p.PackInt64(e.Tmstmp)
	p.PackUint64(e.Hght)
	p.PackID(e.StateRoot)
	p.PackID(e.Destination)
}

func UnmarshalExportBlockMsg(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var export ExportBlockMsg
	p.UnpackID(true, &export.Prnt)
	export.Tmstmp = p.UnpackInt64(true)
	export.Hght = p.UnpackUint64(true)
	p.UnpackID(true, &export.StateRoot)
	p.UnpackID(true, &export.Destination)
	if err := p.Err(); err != nil {
		return nil, err
	}
	return &export, nil
}

func (*ExportBlockMsg) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
