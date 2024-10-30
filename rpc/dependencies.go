// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"
	"github.com/AnomalyFi/nodekit-seq/archiver"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/avalanchego/utils/logging"
)

type Controller interface {
	NetworkID() uint32
	ChainID() ids.ID
	Genesis() *genesis.Genesis
	Tracer() trace.Tracer
	GetTransaction(context.Context, ids.ID) (bool, int64, bool, fees.Dimensions, uint64, error)
	GetBalanceFromState(context.Context, codec.Address) (uint64, error)
	GetRegisteredAnchorsFromState(context.Context) ([][]byte, []*hactions.RollupInfo, error)
	GetEpochExitsFromState(ctx context.Context, epoch uint64) (*storage.EpochExitInfo, error)
	GetArcadiaBuilderFromState(ctx context.Context, epoch uint64) ([]byte, error)
	UnitPrices(ctx context.Context) (fees.Dimensions, error)
	NameSpacesPrice(ctx context.Context, namespaces []string) ([]uint64, error)
	GetAcceptedBlockWindow() int
	Submit(
		ctx context.Context,
		verifySig bool,
		txs []*chain.Transaction,
	) (errs []error)
	Logger() logging.Logger
	// TODO: update this to only have read permission
	Archiver() *archiver.ORMArchiver
}
