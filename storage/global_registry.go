package storage

import (
	"context"
	"encoding/binary"

	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
)

func GlobalRollupRegistryKey() []byte {
	k := make([]byte, 1+consts.Uint16Len)
	k[0] = GlobalRollupRegistryPrefix
	binary.BigEndian.PutUint16(k[1:], GlobalRollupRegistryChunks)
	return k
}

func GetGlobalRollupRegistry(
	ctx context.Context,
	im state.Immutable,
) ([][]byte, error) {
	k := GlobalRollupRegistryKey()
	v, err := im.GetValue(ctx, k)

	if err == database.ErrNotFound {
		return nil, nil
	}
	if err != nil && err != database.ErrNotFound {
		return nil, err
	}
	namespaces, err := UnpackNamespaces(v)
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func SetGlobalRollupRegistry(
	ctx context.Context,
	mu state.Mutable,
	namespaces [][]byte,
) error {
	k := GlobalRollupRegistryKey()
	v, err := PackNamespaces(namespaces)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, v)
}
