// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/nodekit-seq/archiver"
	"github.com/AnomalyFi/nodekit-seq/genesis"
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
) (bool, int64, bool, chain.Dimensions, uint64, error) {
	return storage.GetTransaction(ctx, c.metaDB, txID)
}

func (c *Controller) GetAssetFromState(
	ctx context.Context,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, ed25519.PublicKey, bool, error) {
	return storage.GetAssetFromState(ctx, c.inner.ReadState, asset)
}

func (c *Controller) GetBalanceFromState(
	ctx context.Context,
	pk ed25519.PublicKey,
	asset ids.ID,
) (uint64, error) {
	return storage.GetBalanceFromState(ctx, c.inner.ReadState, pk, asset)
}

func (c *Controller) GetLoanFromState(
	ctx context.Context,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	return storage.GetLoanFromState(ctx, c.inner.ReadState, asset, destination)
}

func (c *Controller) GetBlockFromArchiver(
	ctx context.Context,
	dbBlock *archiver.DBBlock,
) (*chain.StatefulBlock, *ids.ID, error) {
	return c.archiver.GetBlock(dbBlock, c.inner)
}

func (c *Controller) GetByHeight(
	height uint64,
	end int64,
	reply *types.BlockHeadersResponse,
) error {
	return c.archiver.GetByHeight(height, end, c.inner, reply)
}

func (c *Controller) GetByID(
	args *types.GetBlockHeadersIDArgs,
	reply *types.BlockHeadersResponse,
) error {
	return c.archiver.GetByID(args, reply, c.inner)
}

func (c *Controller) GetByStart(
	args *types.GetBlockHeadersByStartArgs,
	reply *types.BlockHeadersResponse,
) error {
	return c.archiver.GetByStart(args, reply, c.inner)
}

func (c *Controller) GetByCommitment(
	args *types.GetBlockCommitmentArgs,
	reply *types.SequencerWarpBlockResponse,
) error {
	return c.archiver.GetByCommitment(args, reply, c.inner)
}
