// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"

	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
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

func (c *Controller) GetAssetFromState(
	ctx context.Context,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, codec.Address, error) {
	return storage.GetAssetFromState(ctx, c.inner.ReadState, asset)
}

func (c *Controller) GetBalanceFromState(
	ctx context.Context,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	return storage.GetBalanceFromState(ctx, c.inner.ReadState, addr, asset)
}

func (c *Controller) GetDataOfStorageSlotFromState(
	ctx context.Context,
	contractAddress ids.ID,
	slot []byte,
) ([]byte, error) {
	return storage.GetBytesFromState(ctx, c.inner.ReadState, contractAddress, slot)
}

func (c *Controller) GetContractFromState(
	ctx context.Context,
	contractAddress ids.ID,
) ([]byte, error) {
	return storage.GetContractFromState(ctx, c.inner.ReadState, contractAddress)
}

func (c *Controller) GetAcceptedBlockWindow() int {
	return c.config.GetAcceptedBlockWindow()
}

func (c *Controller) WarpSigner() warp.Signer {
	return c.snowCtx.WarpSigner
}

func (c *Controller) MessageNetPort() string {
	return c.config.MessageNetPort
}
