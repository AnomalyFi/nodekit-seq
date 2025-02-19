// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"

	"github.com/AnomalyFi/nodekit-seq/storage"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
)

var _ (chain.StateManager) = (*StateManager)(nil)

type StateManager struct{}

func (*StateManager) HeightKey() []byte {
	return storage.HeightKey()
}

func (*StateManager) TimestampKey() []byte {
	return storage.TimestampKey()
}

func (*StateManager) FeeKey() []byte {
	return storage.FeeKey()
}

func (*StateManager) FeeMarketKey() (k []byte) {
	return storage.FeeMarketKey()
}

func (*StateManager) SponsorStateKeys(addr codec.Address) state.Keys {
	return state.Keys{
		string(storage.BalanceKey(addr)): state.Read | state.Write,
	}
}

func (*StateManager) CanDeduct(
	ctx context.Context,
	addr codec.Address,
	im state.Immutable,
	amount uint64,
) error {
	bal, err := storage.GetBalance(ctx, im, addr)
	if err != nil {
		return err
	}
	if bal < amount {
		return storage.ErrInvalidBalance
	}
	return nil
}

func (*StateManager) Deduct(
	ctx context.Context,
	addr codec.Address,
	mu state.Mutable,
	amount uint64,
) error {
	return storage.SubBalance(ctx, mu, addr, amount)
}
