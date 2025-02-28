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

func RollupInfoKey(namespace []byte) []byte {
	return hactions.RollupInfoKey(namespace)
}

func GetRollupInfo(
	ctx context.Context,
	im state.Immutable,
	namespace []byte,
) (*hactions.RollupInfo, error) {
	k := RollupInfoKey(namespace)
	rollup, _, err := innerGetRollupInfo(im.GetValue(ctx, k))
	return rollup, err
}

// Used to serve RPC queries
func GetRollupInfoFromState(
	ctx context.Context,
	f ReadState,
	namespace []byte,
) (*hactions.RollupInfo, error) {
	k := RollupInfoKey(namespace)
	values, errs := f(ctx, [][]byte{k})
	rollup, _, err := innerGetRollupInfo(values[0], errs[0])
	return rollup, err
}

func innerGetRollupInfo(
	v []byte,
	err error,
) (*hactions.RollupInfo, bool, error) { //nolint:unparam
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p := codec.NewReader(v, consts.NetworkSizeLimit)
	info, err := hactions.UnmarshalRollupInfo(p)
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func SetRollupInfo(
	ctx context.Context,
	mu state.Mutable,
	namespace []byte,
	info *hactions.RollupInfo,
) error {
	k := RollupInfoKey(namespace)
	p := codec.NewWriter(info.Size(), consts.NetworkSizeLimit)
	info.Marshal(p)
	if err := p.Err(); err != nil {
		return err
	}
	infoBytes := p.Bytes()
	return mu.Insert(ctx, k, infoBytes)
}

func DelRollupInfo(
	ctx context.Context,
	mu state.Mutable,
	namespace []byte,
) error {
	k := RollupInfoKey(namespace)
	return mu.Remove(ctx, k)
}
