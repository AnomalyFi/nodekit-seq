package storage

import (
	"context"
	"errors"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func EpochExitsKey(epoch uint64) []byte {
	return hactions.EpochExitsKey(epoch)
}

// This should get all the exits for 1 epoch
func GetEpochExits(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) (*hactions.EpochExitInfo, bool, error) {
	_, ep, exists, err := getEpochExits(ctx, im, epoch)
	return ep, exists, err
}

func getEpochExits(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) ([]byte, *hactions.EpochExitInfo, bool, error) {
	k := EpochExitsKey(epoch)
	epochExit, exists, err := innerGetEpochExits(im.GetValue(ctx, k))
	return k, epochExit, exists, err
}

// Used to serve RPC queries
func GetEpochExitsFromState(
	ctx context.Context,
	f ReadState,
	epoch uint64,
) (*hactions.EpochExitInfo, error) {
	k := EpochExitsKey(epoch)
	values, errs := f(ctx, [][]byte{k})
	epochExit, _, err := innerGetEpochExits(values[0], errs[0])
	return epochExit, err
}

func innerGetEpochExits(
	v []byte,
	err error,
) (*hactions.EpochExitInfo, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p := codec.NewReader(v, consts.NetworkSizeLimit)
	info, err := hactions.UnmarshalEpochExitsInfo(p)
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func SetEpochExits(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
	info *hactions.EpochExitInfo,
) error {
	k := EpochExitsKey(epoch)
	return setEpochExits(ctx, mu, k, info)
}

func setEpochExits(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	info *hactions.EpochExitInfo,
) error {
	var size int
	for _, e := range info.Exits {
		size += e.Size()
	}

	p := codec.NewWriter(size, consts.NetworkSizeLimit)
	err := info.Marshal(p)
	if err != nil {
		return err
	}
	infoBytes := p.Bytes()
	return mu.Insert(ctx, key, infoBytes)
}
