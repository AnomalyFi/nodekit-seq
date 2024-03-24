// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	"github.com/anomalyFi/nodekit-seq/cmd/token-feed/manager"
	"github.com/ava-labs/hypersdk/codec"
)

type Manager interface {
	GetFeedInfo(context.Context) (codec.Address, uint64, error)
	GetFeed(context.Context) ([]*manager.FeedObject, error)
}
