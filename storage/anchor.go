package storage

import (
	"context"
	"errors"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func AnchorRegistryKey() []byte {
	return hactions.AnchorRegistryKey()
}

func GetAnchorRegistry(
	ctx context.Context,
	im state.Immutable,
) ([][]byte, error) {
	k := AnchorRegistryKey()
	namespaces, _, err := innerGetRegistry(im.GetValue(ctx, k))
	return namespaces, err
}

// Used to serve RPC queries
func GetAnchorsFromState(
	ctx context.Context,
	f ReadState,
) ([][]byte, []*hactions.RollupInfo, error) {
	k := AnchorRegistryKey()
	values, errs := f(ctx, [][]byte{k})
	namespaces, _, err := innerGetRegistry(values[0], errs[0])
	if err != nil {
		return nil, nil, err
	}

	infos := make([]*hactions.RollupInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		k := RollupInfoKey(ns)
		values, errs := f(ctx, [][]byte{k})
		info, _, err := innerGetRollupInfo(values[0], errs[0])
		if err != nil {
			return nil, nil, err
		}
		infos = append(infos, info)
	}
	return namespaces, infos, err
}

func innerGetRegistry(
	v []byte,
	err error,
) ([][]byte, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	namespaces, err := UnpackNamespaces(v)
	if err != nil {
		return nil, false, err
	}

	return namespaces, true, nil
}

func SetAnchorRegistry(
	ctx context.Context,
	mu state.Mutable,
	namespaces [][]byte,
) error {
	k := AnchorRegistryKey()
	packed, err := PackNamespaces(namespaces)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, packed)
}
