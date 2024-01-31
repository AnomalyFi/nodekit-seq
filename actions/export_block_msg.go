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
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/state"
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

func (*ExportBlockMsg) GetTypeID() uint8 {
	return exportBlockID
}

func (e *ExportBlockMsg) StateKeys(rauth chain.Auth, _ ids.ID) []string {
	// TODO may need to be fixed
	return []string{
		string(storage.PrefixBlockKey(e.StateRoot, e.Destination)),
		string(storage.PrefixBlockKey(e.Prnt, e.Destination)),
	}
}

// TODO probably need to fix
func (e *ExportBlockMsg) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.AssetChunks, storage.LoanChunks, storage.BalanceChunks}
}

func (e *ExportBlockMsg) executeLoan(
	ctx context.Context,
	mu state.Mutable,
	actor ed25519.PublicKey,
	txID ids.ID,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	// TODO may need to add a destination chainID to this
	wt := &chain.WarpBlock{
		Prnt:      e.Prnt,
		Tmstmp:    e.Tmstmp,
		Hght:      e.Hght,
		StateRoot: e.StateRoot,
		TxID:      txID,
	}
	payload, err := wt.Marshal()
	if err != nil {
		return false, ExportBlockComputeUnits, utils.ErrBytes(err), nil, nil
	}
	wm := &warp.UnsignedMessage{
		// NetworkID + SourceChainID is populated by hypersdk
		Payload: payload,
	}
	return true, ExportBlockComputeUnits, nil, wm, nil
}

func (e *ExportBlockMsg) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	rauth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	actor := auth.GetActor(rauth)
	// unitsUsed := e.MaxUnits(r) // max units == units
	if e.Destination == ids.Empty {
		// This would result in multiplying balance export by whoever imports the
		// transaction.
		return false, ExportBlockComputeUnits, OutputAnycast, nil, nil
	}
	// TODO: check if destination is ourselves
	// if e.Return {
	// 	return e.executeReturn(ctx, r, db, actor, txID)
	// }
	return e.executeLoan(ctx, mu, actor, txID)
}

func (*ExportBlockMsg) MaxUnits(chain.Rules) uint64 {
	// TODO fix this
	return ExportBlockComputeUnits
}

func (*ExportBlockMsg) Size() int {
	return consts.IDLen + consts.Int64Len + consts.Uint64Len + consts.Int64Len + consts.IDLen
}

func (*ExportBlockMsg) OutputsWarpMessage() bool {
	return true
}

func (*ExportBlockMsg) MaxComputeUnits(chain.Rules) uint64 {
	return ExportBlockComputeUnits
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

func (*ExportBlockMsg) NMTNamespace() []byte {
	// byte array with 8 zeros
	return DefaultNMTNamespace
}
