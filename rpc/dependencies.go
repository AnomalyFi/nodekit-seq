// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/avalanchego/utils/logging"
)

type Controller interface {
	Genesis() *genesis.Genesis
	Tracer() trace.Tracer
	GetTransaction(context.Context, ids.ID) (bool, int64, bool, fees.Dimensions, uint64, error)
	GetAssetFromState(context.Context, ids.ID) (bool, []byte, uint8, []byte, uint64, codec.Address, error)
	GetBalanceFromState(context.Context, codec.Address, ids.ID) (uint64, error)
	GetDataOfStorageSlotFromState(ctx context.Context, contractAddress ids.ID, slot string) ([]byte, error)
	UnitPrices(ctx context.Context) (fees.Dimensions, error)
	GetAcceptedBlockWindow() int
	Submit(
		ctx context.Context,
		verifySig bool,
		txs []*chain.Transaction,
	) (errs []error)
	Logger() logging.Logger
	MessageNetPort() string
}
