// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/ava-labs/avalanchego/ids"
)

type Manager interface {
	GetFaucetAddress(context.Context) (ed25519.PublicKey, error)
	GetChallenge(context.Context) ([]byte, uint16, error)
	SolveChallenge(context.Context, ed25519.PublicKey, []byte, []byte) (ids.ID, uint64, error)
}
