package actions

import (
	"bytes"
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
)

// checks if the actor is in the list of whitelisted addresses.
// whitelisted addresses are allowed to submit auction transactions.
func isWhiteListed(rules chain.Rules, actor codec.Address) bool {
	whitelistedAddressesB, _ := rules.FetchCustom("whitelisted.Addresses")
	whitelistedAddresses := whitelistedAddressesB.([]codec.Address)
	return containsAddress(whitelistedAddresses, actor)
}

// checks if a given address is in the list of addresses.
func containsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

// checks if a given byte slice is in the list of byte slices.
func contains(arr [][]byte, ns []byte) bool {
	for _, v := range arr {
		if bytes.Equal(v, ns) {
			return true
		}
	}
	return false
}

// ArcadiaFundAddress returns the address of Arcadia fund.
func ArcadiaFundAddress() codec.Address {
	b := make([]byte, 33)
	b[32] = 0x1
	addr, _ := codec.ToAddress(b)
	return addr
}

// checks if ns is in the list of registered namespaces.
// fetches the info of the rollup and checks if the actor is the authority.
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

// Epoch returns the epoch number for a given block height.
func Epoch(blockHeight uint64, epochLength int64) uint64 {
	return blockHeight / uint64(epochLength)
}
