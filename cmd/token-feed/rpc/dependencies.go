// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/nodekit-seq/cmd/token-feed/manager"
)

type Manager interface {
	GetFeedInfo(context.Context) (ed25519.PublicKey, uint64, error)
	GetFeed(context.Context) ([]*manager.FeedObject, error)
}
