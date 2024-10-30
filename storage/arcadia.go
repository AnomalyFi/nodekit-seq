package storage

import (
	"context"
	"encoding/binary"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
)

func ArcadiaRegistryKey() []byte {
	return hactions.ArcadiaRegistryKey()
}

func GetArcadiaRegistry(
	ctx context.Context,
	im state.Immutable,
) ([][]byte, error) {
	k := ArcadiaRegistryKey()
	namespaces, _, err := innerGetRegistry(im.GetValue(ctx, k))
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func SetArcadiaRegistry(
	ctx context.Context,
	mu state.Mutable,
	namespaces [][]byte,
) error {
	k := ArcadiaRegistryKey()
	packed, err := PackNamespaces(namespaces)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, packed)
}

func ArcadiaBidKey(epoch uint64) []byte {
	k := make([]byte, 1+8+consts.Uint16Len)
	k[0] = ArcadiaBidPrefix
	binary.BigEndian.PutUint64(k[1:], epoch)
	binary.BigEndian.PutUint16(k[9:], EpochExitChunks)
	return k
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
	v := make([]byte, 8+48+96)
	binary.BigEndian.PutUint64(v[:8], bidPrice)
	copy(v[8:], bidderPublicKey)
	copy(v[8+48:], bidderSignature)
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

func GetArcadiaBidderInfoAndSignature(
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
	if errs[0] != nil {
		return nil, errs[0]
	}
	return values[0][8 : 8+48], nil
}
