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
	rollupregistry "github.com/AnomalyFi/nodekit-seq/rollup_registry"
	"github.com/AnomalyFi/nodekit-seq/types"
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
	GetEpochExitsFromState(ctx context.Context, epoch uint64) (*hactions.EpochExitInfo, error)
	GetBuilderFromState(ctx context.Context, epoch uint64) ([]byte, error)
	GetRollupInfoFromState(ctx context.Context, namespace []byte) (*hactions.RollupInfo, error)
	GetRollupRegistryFromState(ctx context.Context) ([][]byte, error)
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
	RollupRegistry() *rollupregistry.RollupRegistry
	GetHighestSettledToBNonceFromState(ctx context.Context) (uint64, error)
	GetCertByChunkIDFromState(ctx context.Context, chunkID ids.ID) (*types.DACertInfo, error)
	GetCertByChainInfoFromState(ctx context.Context, chainID string, blockNumber uint64) (*types.DACertInfo, error)
	GetCertsByToBNonceFromState(ctx context.Context, tobNonce uint64) ([]*types.DACertInfo, error)
	GetLowestToBNonceAtEpochFromState(ctx context.Context, epoch uint64) (uint64, error)
}
