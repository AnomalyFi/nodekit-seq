package storage

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func EpochExitsKey(epoch uint64) []byte {
	k := make([]byte, 1+8+consts.Uint16Len)
	k[0] = EpochExitsPrefix
	binary.BigEndian.PutUint64(k[1:], epoch)
	binary.BigEndian.PutUint16(k[9:], EpochExitsChunks)
	return k
}

// This should get all the exits for 1 epoch
func GetEpochExits(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) (*EpochExitInfo, bool, error) {
	_, ep, exists, err := getEpochExits(ctx, im, epoch)
	return ep, exists, err
}

func getEpochExits(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) ([]byte, *EpochExitInfo, bool, error) {
	k := EpochExitsKey(epoch)
	epochExit, exists, err := innerGetEpochExits(im.GetValue(ctx, k))
	return k, epochExit, exists, err
}

// Used to serve RPC queries
func GetEpochExitsFromState(
	ctx context.Context,
	f ReadState,
	epoch uint64,
) (*EpochExitInfo, error) {
	k := EpochExitsKey(epoch)
	values, errs := f(ctx, [][]byte{k})
	epochExit, _, err := innerGetEpochExits(values[0], errs[0])
	return epochExit, err
}

func innerGetEpochExits(
	v []byte,
	err error,
) (*EpochExitInfo, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p := codec.NewReader(v, consts.NetworkSizeLimit)
	info, err := UnmarshalEpochExitsInfo(p)
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func SetEpochExits(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
	info *EpochExitInfo,
) error {
	k := EpochExitsKey(epoch)
	return setEpochExits(ctx, mu, k, info)
}

func setEpochExits(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	info *EpochExitInfo,
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
