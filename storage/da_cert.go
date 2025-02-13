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
	binary.BigEndian.PutUint16(k[1:], DACertficateToBNonceChunks)
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

// for mapping epoch to the lowest tobnonce in that epoch, this can give a rough range of chunks
// for rollups/sidecar to traverse through at derivation pipeline
// for highest stored DACert ToBNonce
func ToBNonceEpochKey(epoch uint64) []byte {
	k := make([]byte, 1+consts.Uint64Len+consts.Uint16Len)
	k[0] = EpochToBNoncePrefix
	binary.BigEndian.PutUint64(k[1:], epoch)
	binary.BigEndian.PutUint16(k[1+consts.Uint64Len:], EpochToBNonceChunks)
	return k
}

func SetToBNonceAtEpoch(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
	tobNonce uint64,
) error {
	k := ToBNonceEpochKey(epoch)
	_, exists, err := innerGetBalance(mu.GetValue(ctx, k))
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return setBalance(ctx, mu, k, tobNonce)
}

func GetToBNonceAtEpoch(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) (uint64, error) {
	k := ToBNonceEpochKey(epoch)
	nonce, _, err := innerGetBalance(im.GetValue(ctx, k))
	return nonce, err
}

// Used to serve RPC queries
func GetToBNonceAtEpochFromState(
	ctx context.Context,
	f ReadState,
	epoch uint64,
) (uint64, error) {
	k := ToBNonceEpochKey(epoch)
	values, errs := f(ctx, [][]byte{k})
	nonce, exists, err := innerGetBalance(values[0], errs[0])
	if !exists {
		return 0, ErrToBNonceNotExistsForEpoch
	}
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
	switch err {
	case database.ErrNotFound:
		chunkIDs = make([]ids.ID, 0)
	case nil:
		chunkIDs, err = UnpackIDs(value)
		if err != nil {
			return err
		}
	default:
		return err
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
func DACertByChunkIDKey(chunkID ids.ID) []byte {
	chunkIDStr := chunkID.String()
	k := make([]byte, 1+len(chunkIDStr)+consts.Uint16Len)
	k[0] = DACertPrefix
	copy(k[1:1+len(chunkIDStr)], []byte(chunkIDStr))
	binary.BigEndian.PutUint16(k[1+len(chunkIDStr):], DACertificateChunks)
	return k
}

func GetDACertByChunkID(
	ctx context.Context,
	im state.Immutable,
	chunkID ids.ID,
) (*types.DACertInfo, error) {
	k := DACertByChunkIDKey(chunkID)
	value, err := im.GetValue(ctx, k)
	if err != nil {
		return nil, err
	}

	p := codec.NewReader(value, types.CertInfoSizeLimit)
	return types.UnmarshalCertInfo(p)
}

// Used to serve RPC queries
func GetDACertByChunkIDFromState(
	ctx context.Context,
	f ReadState,
	chunkID ids.ID,
) (*types.DACertInfo, error) {
	k := DACertByChunkIDKey(chunkID)
	values, errs := f(ctx, [][]byte{k})
	if errors.Is(errs[0], database.ErrNotFound) {
		return nil, types.ErrCertNotExists
	}
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
	k := DACertByChunkIDKey(chunkID)
	p := codec.NewWriter(128, types.CertInfoSizeLimit)
	cert.Marshal(p)
	if p.Err() != nil {
		return p.Err()
	}
	raw := p.Bytes()
	return mu.Insert(ctx, k, raw)
}

// for mapping <chainID, blockNumber> to chunkID
func DACertChunkIDKey(chainID string, blockNumber uint64) []byte {
	k := make([]byte, 1+len(chainID)+consts.Uint64Len+consts.Uint16Len)
	k[0] = DACertChunkIDPrefix
	copy(k[1:], []byte(chainID))
	binary.BigEndian.PutUint64(k[1+len(chainID):], blockNumber)
	binary.BigEndian.PutUint16(k[1+len(chainID)+consts.Uint64Len:], DACertificateChunkIDChunks)
	return k
}

func GetDACertChunkID(
	ctx context.Context,
	im state.Immutable,
	chainID string,
	blockNumber uint64,
) (ids.ID, error) {
	ret := ids.ID{}
	k := DACertChunkIDKey(chainID, blockNumber)
	value, err := im.GetValue(ctx, k)
	if err != nil {
		return ids.Empty, err
	}

	p := codec.NewReader(value, ids.IDLen)
	p.UnpackID(true, &ret)
	return ret, p.Err()
}

// Used to serve RPC queries
func GetDACertChunkIDFromState(
	ctx context.Context,
	f ReadState,
	chainID string,
	blockNumber uint64,
) (ids.ID, error) {
	ret := ids.ID{}
	k := DACertChunkIDKey(chainID, blockNumber)
	values, errs := f(ctx, [][]byte{k})
	if errors.Is(errs[0], database.ErrNotFound) {
		return ids.Empty, types.ErrCertChunkIDNotExists
	}
	if errs[0] != nil {
		return ids.Empty, errs[0]
	}
	p := codec.NewReader(values[0], ids.IDLen)
	p.UnpackID(true, &ret)
	return ret, p.Err()
}

func SetDACertChunkID(
	ctx context.Context,
	mu state.Mutable,
	chainID string,
	blockNumber uint64,
	chunkID ids.ID,
) error {
	k := DACertChunkIDKey(chainID, blockNumber)
	p := codec.NewWriter(ids.IDLen, ids.IDLen)
	p.PackID(chunkID)
	if err := p.Err(); err != nil {
		return err
	}
	raw := p.Bytes()
	return mu.Insert(ctx, k, raw)
}
