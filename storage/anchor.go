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

func AnchorRegistryKey() []byte {
	return hactions.AnchorRegistryKey()
}

func PackNamespaces(namespaces [][]byte) ([]byte, error) {
	return hactions.PackNamespaces(namespaces)
}

func UnpackNamespaces(raw []byte) ([][]byte, error) {
	return hactions.UnpackNamespaces(raw)
}

func GetAnchors(
	ctx context.Context,
	im state.Immutable,
) ([][]byte, []*hactions.AnchorInfo, error) {
	_, namespaces, infos, _, err := getAnchors(ctx, im)
	return namespaces, infos, err
}

func getAnchors(
	ctx context.Context,
	im state.Immutable,
) ([]byte, [][]byte, []*hactions.AnchorInfo, bool, error) {
	k := AnchorRegistryKey()
	namespaces, exists, err := innerGetAnchors(im.GetValue(ctx, k))
	if err != nil {
		return nil, nil, nil, false, err
	}

	infos := make([]*hactions.AnchorInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		info, err := GetAnchor(ctx, im, ns)
		if err != nil {
			return nil, nil, nil, false, err
		}
		infos = append(infos, info)
	}
	return k, namespaces, infos, exists, err
}

// Used to serve RPC queries
func GetAnchorsFromState(
	ctx context.Context,
	f ReadState,
) ([][]byte, []*hactions.AnchorInfo, error) {
	k := AnchorRegistryKey()
	values, errs := f(ctx, [][]byte{k})
	namespaces, _, err := innerGetAnchors(values[0], errs[0])
	if err != nil {
		return nil, nil, err
	}

	infos := make([]*hactions.AnchorInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		k := AnchorKey(ns)
		values, errs := f(ctx, [][]byte{k})
		info, _, err := innerGetAnchor(values[0], errs[0])
		if err != nil {
			return nil, nil, err
		}
		infos = append(infos, info)
	}
	return namespaces, infos, err
}

func innerGetAnchors(
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

func SetAnchors(
	ctx context.Context,
	mu state.Mutable,
	namespaces [][]byte,
) error {
	return setAnchors(ctx, mu, namespaces)
}

func setAnchors(
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

func AnchorKey(namespace []byte) []byte {
	return hactions.AnchorKey(namespace)
}

// If locked is 0, then account does not exist
func GetAnchor(
	ctx context.Context,
	im state.Immutable,
	namespace []byte,
) (*hactions.AnchorInfo, error) {
	_, anchor, _, err := getAnchor(ctx, im, namespace)
	return anchor, err
}

func getAnchor(
	ctx context.Context,
	im state.Immutable,
	namespace []byte,
) ([]byte, *hactions.AnchorInfo, bool, error) {
	k := AnchorKey(namespace)
	anchor, exists, err := innerGetAnchor(im.GetValue(ctx, k))
	return k, anchor, exists, err
}

// Used to serve RPC queries
func GetAnchorFromState(
	ctx context.Context,
	f ReadState,
	namespace []byte,
) (*hactions.AnchorInfo, error) {
	k := AnchorKey(namespace)
	values, errs := f(ctx, [][]byte{k})
	anchor, _, err := innerGetAnchor(values[0], errs[0])
	return anchor, err
}

func innerGetAnchor(
	v []byte,
	err error,
) (*hactions.AnchorInfo, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p := codec.NewReader(v, consts.NetworkSizeLimit)
	info, err := hactions.UnmarshalAnchorInfo(p)
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func SetAnchor(
	ctx context.Context,
	mu state.Mutable,
	namespace []byte,
	info *hactions.AnchorInfo,
) error {
	k := AnchorKey(namespace)
	return setAnchor(ctx, mu, k, info)
}

func setAnchor(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	info *hactions.AnchorInfo,
) error {
	p := codec.NewWriter(info.Size(), consts.NetworkSizeLimit)
	err := info.Marshal(p)
	if err != nil {
		return err
	}
	infoBytes := p.Bytes()
	return mu.Insert(ctx, key, infoBytes)
}

func DelAnchor(
	ctx context.Context,
	mu state.Mutable,
	namespace []byte,
) error {
	k := AnchorKey(namespace)
	return delAnchor(ctx, mu, k)
}

func delAnchor(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
) error {
	return mu.Remove(ctx, key)
}
