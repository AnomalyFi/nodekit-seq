// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"

	"github.com/AnomalyFi/nodekit-seq/archiver"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	rollupregistry "github.com/AnomalyFi/nodekit-seq/rollup_registry"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/avalanchego/utils/logging"
)

func (c *Controller) Genesis() *genesis.Genesis {
	return c.genesis
}

func (c *Controller) Logger() logging.Logger {
	return c.inner.Logger()
}

func (c *Controller) Tracer() trace.Tracer {
	return c.inner.Tracer()
}

func (c *Controller) GetTransaction(
	ctx context.Context,
	txID ids.ID,
) (bool, int64, bool, fees.Dimensions, uint64, error) {
	return storage.GetTransaction(ctx, c.metaDB, txID)
}

func (c *Controller) GetBalanceFromState(
	ctx context.Context,
	addr codec.Address,
) (uint64, error) {
	return storage.GetBalanceFromState(ctx, c.inner.ReadState, addr)
}

func (c *Controller) GetRollupRegistryFromState(
	ctx context.Context,
) ([][]byte, error) {
	return storage.GetRollupRegistryFromState(ctx, c.inner.ReadState)
}

func (c *Controller) GetEpochExitsFromState(
	ctx context.Context,
	epoch uint64,
) (*hactions.EpochExitInfo, error) {
	info, err := storage.GetEpochExitsFromState(ctx, c.inner.ReadState, epoch)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (c *Controller) GetRollupInfoFromState(
	ctx context.Context,
	namespace []byte,
) (*hactions.RollupInfo, error) {
	return storage.GetRollupInfoFromState(ctx, c.inner.ReadState, namespace)
}

func (c *Controller) GetBuilderFromState(
	ctx context.Context,
	epoch uint64,
) ([]byte, error) {
	return storage.GetArcadiaBuilderFromState(ctx, c.inner.ReadState, epoch)
}

func (c *Controller) GetAcceptedBlockWindow() int {
	return c.config.GetAcceptedBlockWindow()
}

func (c *Controller) Archiver() *archiver.ORMArchiver {
	return c.archiver
}

func (c *Controller) RollupRegistry() *rollupregistry.RollupRegistry {
	return c.rollupRegistry
}

func (c *Controller) NetworkID() uint32 {
	return c.snowCtx.NetworkID
}

func (c *Controller) ChainID() ids.ID {
	return c.snowCtx.ChainID
}

func (c *Controller) GetCertByChunkIDFromState(ctx context.Context, chunkID ids.ID) (*types.DACertInfo, error) {
	return storage.GetDACertByChunkIDFromState(ctx, c.inner.ReadState, chunkID)
}

func (c *Controller) GetCertByChainInfoFromState(ctx context.Context, chainID string, blockNumber uint64) (*types.DACertInfo, error) {
	chunkID, err := storage.GetDACertChunkIDFromState(ctx, c.inner.ReadState, chainID, blockNumber)
	if err != nil {
		return nil, err
	}
	return storage.GetDACertByChunkIDFromState(ctx, c.inner.ReadState, chunkID)
}

func (c *Controller) GetCertsByToBNonceFromState(ctx context.Context, tobNonce uint64) ([]*types.DACertInfo, error) {
	chunkIDs, err := storage.GetDACertChunkIDsFromState(ctx, c.inner.ReadState, tobNonce)
	if err != nil {
		return nil, fmt.Errorf("cannot get chunkIDs from state: %w", err)
	}

	ret := make([]*types.DACertInfo, 0, len(chunkIDs))
	for i := 0; i < len(chunkIDs); i++ {
		cert, err := storage.GetDACertByChunkIDFromState(ctx, c.inner.ReadState, chunkIDs[i])
		if err != nil {
			return nil, fmt.Errorf("cannot get cert by chunkID: %s: %w", chunkIDs[i].String(), err)
		}
		ret = append(ret, cert)
	}
	return ret, nil
}

func (c *Controller) GetLowestToBNonceAtEpochFromState(ctx context.Context, epoch uint64) (uint64, error) {
	return storage.GetToBNonceAtEpochFromState(ctx, c.inner.ReadState, epoch)
}
