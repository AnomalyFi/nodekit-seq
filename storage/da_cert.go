package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"slices"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/types"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

// for highest stored DACert ToBNonce
func DACertToBNonceKey() []byte {
	k := make([]byte, 1+consts.Uint16Len)
	k[0] = DACertToBNoncePrefix
	binary.BigEndian.PutUint16(k[1+consts.Uint64Len:], DACertficateToBNonceChunks)
	return k
}

func SetDACertToBNonce(
	ctx context.Context,
	mu state.Mutable,
	tobNonce uint64,
) error {
	k := DACertToBNonceKey()
	return setBalance(ctx, mu, k, tobNonce)
}

func GetDACertToBNonce(
	ctx context.Context,
	im state.Immutable,
) (uint64, error) {
	k := DACertToBNonceKey()
	nonce, _, err := innerGetBalance(im.GetValue(ctx, k))
	return nonce, err
}

// Used to serve RPC queries
func GetDAToBNonceFromState(
	ctx context.Context,
	f ReadState,
) (uint64, error) {
	k := DACertToBNonceKey()
	values, errs := f(ctx, [][]byte{k})
	nonce, _, err := innerGetBalance(values[0], errs[0])
	return nonce, err
}

// for cert indexes
func DACertIndexKey(tobNonce uint64) []byte {
	k := make([]byte, 1+consts.Uint64Len+consts.Uint16Len)
	k[0] = DACertIndexPrefix
	binary.BigEndian.PutUint64(k[1:], tobNonce)
	binary.BigEndian.PutUint16(k[1+consts.Uint64Len:], DACertficateIndexChunks)
	return k
}

func AddDACertToLayer(
	ctx context.Context,
	mu state.Mutable,
	tobNonce uint64,
	chunkID ids.ID,
) error {
	k := DACertIndexKey(tobNonce)
	var chunkIDs []ids.ID
	value, err := mu.GetValue(ctx, k)
	if errors.Is(err, database.ErrNotFound) {
		chunkIDs = make([]ids.ID, 0)
	} else if err != nil {
		return err
	} else {
		chunkIDs, err = UnpackIDs(value)
		if err != nil {
			return err
		}
	}
	if slices.ContainsFunc(chunkIDs, func(c ids.ID) bool {
		return chunkID.Compare(c) == 0
	}) {
		return ErrCertExists
	}

	chunkIDs = append(chunkIDs, chunkID)
	raw, err := PackIDs(chunkIDs)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, raw)
}

func GetDACertChunkIDs(
	ctx context.Context,
	im state.Immutable,
	tobNonce uint64,
) ([]ids.ID, error) {
	k := DACertIndexKey(tobNonce)
	value, err := im.GetValue(ctx, k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return UnpackIDs(value)
}

// Used to serve RPC queries
func GetDACertChunkIDsFromState(
	ctx context.Context,
	f ReadState,
	tobNonce uint64,
) ([]ids.ID, error) {
	k := DACertIndexKey(tobNonce)
	values, errs := f(ctx, [][]byte{k})
	err := errs[0]
	if errors.Is(err, database.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return UnpackIDs(values[0])
}

// for individual cert
func DACertKey(chunkID ids.ID) []byte {
	k := make([]byte, 1+ids.IDLen+consts.Uint16Len)
	k[0] = DACertPrefix
	copy(k[1:], chunkID[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen:], DACertificateChunks)
	return k
}

func GetDACert(
	ctx context.Context,
	im state.Immutable,
	chunkID ids.ID,
) (*types.DACertInfo, error) {
	k := DACertKey(chunkID)
	value, err := im.GetValue(ctx, k)
	if err != nil {
		return nil, err
	}

	p := codec.NewReader(value, types.CertInfoSizeLimit)
	return types.UnmarshalCertInfo(p)
}

// Used to serve RPC queries
func GetDACertFromState(
	ctx context.Context,
	f ReadState,
	chunkID ids.ID,
) (*types.DACertInfo, error) {
	k := DACertKey(chunkID)
	values, errs := f(ctx, [][]byte{k})
	if errs[0] != nil {
		return nil, errs[0]
	}
	p := codec.NewReader(values[0], types.CertInfoSizeLimit)
	return types.UnmarshalCertInfo(p)
}

func SetDACert(
	ctx context.Context,
	mu state.Mutable,
	chunkID ids.ID,
	cert *types.DACertInfo,
) error {
	k := DACertKey(chunkID)
	p := codec.NewWriter(128, types.CertInfoSizeLimit)
	if err := cert.Marshal(p); err != nil {
		return err
	}
	raw := p.Bytes()
	return mu.Insert(ctx, k, raw)
}
