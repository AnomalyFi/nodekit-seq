package storage

import (
	"context"
	"errors"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func RollupRegistryKey() []byte {
	return hactions.RollupRegistryKey()
}

func GetRollupRegistry(
	ctx context.Context,
	im state.Immutable,
) ([][]byte, error) {
	k := RollupRegistryKey()
	namespaces, _, err := innerGetRegistry(im.GetValue(ctx, k))
	return namespaces, err
}

// Used to serve RPC queries
func GetRollupRegistryFromState(
	ctx context.Context,
	f ReadState,
) ([][]byte, error) {
	k := RollupRegistryKey()
	values, errs := f(ctx, [][]byte{k})
	namespaces, _, err := innerGetRegistry(values[0], errs[0])
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func innerGetRegistry(
	v []byte,
	err error,
) ([][]byte, bool, error) { //nolint:unparam
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

func SetRollupRegistry(
	ctx context.Context,
	mu state.Mutable,
	namespaces [][]byte,
) error {
	k := RollupRegistryKey()
	packed, err := PackNamespaces(namespaces)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, packed)
}
