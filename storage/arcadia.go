package storage

import (
	"context"
	"encoding/binary"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func ArcadiaBidKey(epoch uint64) []byte {
	return hactions.ArcadiaBidKey(epoch)
}

func StoreArcadiaBidInfo(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
	bidPrice uint64,
	bidderPublicKey []byte, // 48 bytes
	bidderSignature []byte, // 96 bytes
) error {
	k := ArcadiaBidKey(epoch)
	v := hactions.PackArcadiaAuctionWinner(bidPrice, bidderPublicKey, bidderSignature)
	return mu.Insert(ctx, k, v)
}

func GetArcadiaBidValue(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) (uint64, error) {
	k := ArcadiaBidKey(epoch)
	v, err := im.GetValue(ctx, k)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(v[:8]), nil
}

func GetArcadiaBuilderInfoAndSignature(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) ([]byte, []byte, error) {
	k := ArcadiaBidKey(epoch)
	v, err := im.GetValue(ctx, k)
	if err != nil {
		return nil, nil, err
	}
	return v[8 : 8+48], v[8+48:], nil
}

func GetArcadiaBuilderFromState(
	ctx context.Context,
	f ReadState,
	epoch uint64,
) ([]byte, error) {
	k := ArcadiaBidKey(epoch)
	values, errs := f(ctx, [][]byte{k})
	if errs[0] == database.ErrNotFound {
		return nil, nil
	}
	if errs[0] != nil {
		return nil, errs[0]
	}
	return values[0][8 : 8+48], nil
}
