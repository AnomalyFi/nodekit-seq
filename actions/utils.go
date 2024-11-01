package actions

import (
	"bytes"
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
)

func isWhiteListed(rules chain.Rules, actor codec.Address) bool {
	whitelistedAddressesB, _ := rules.FetchCustom("whitelisted.Addresses")
	whitelistedAddresses := whitelistedAddressesB.([]codec.Address)
	return containsAddress(whitelistedAddresses, actor)
}

func containsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

func contains(arr [][]byte, ns []byte) bool {
	for _, v := range arr {
		if bytes.Equal(v, ns) {
			return true
		}
	}
	return false
}

func ArcadiaFundAddress() codec.Address {
	b := make([]byte, 33)
	b[32] = 0x1
	addr, _ := codec.ToAddress(b)
	return addr
}

func authorizationChecks(ctx context.Context, actor codec.Address, namespaces [][]byte, ns []byte, im state.Immutable) error {
	if !contains(namespaces, ns) {
		return ErrNameSpaceNotRegistered
	}

	info, err := storage.GetRollupInfo(ctx, im, ns)
	if err != nil {
		return err
	}

	if info.AuthoritySEQAddress != actor {
		return ErrNotAuthorized
	}

	return nil
}

func Epoch(blockHeight uint64, epochLength int64) uint64 {
	return uint64(int64(blockHeight) / epochLength)
}
