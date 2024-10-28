// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"
	tconsts "github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	smath "github.com/ava-labs/avalanchego/utils/math"
)

type ReadState func(context.Context, [][]byte) ([][]byte, []error)

// Metadata
// 0x0/ (tx)
//   -> [txID] => timestamp
//
// State
// 0x0/ (balance)
//   -> [owner|asset] => balance
// 0x1/ (assets)
//   -> [asset] => metadataLen|metadata|supply|owner
// 0x2/ (orders)
//   -> [txID] => in|out|rate|remaining|owner
// 0x3/ (loans)
//   -> [assetID|destination] => amount
// 0x4/ (hypersdk-height)
// 0x5/ (hypersdk-timestamp)
// 0x6/ (hypersdk-fee)
// 0x7/ (hypersdk-incoming warp)
// 0x8/ (hypersdk-outgoing warp)

const (
	// metaDB
	txPrefix = 0x0

	// TODO: clean up the prefixes below
	// stateDB
	balancePrefix           = 0x0
	assetPrefix             = 0x1
	orderPrefix             = 0x2
	loanPrefix              = 0x3
	heightPrefix            = 0x4
	timestampPrefix         = 0x5
	feePrefix               = 0x6
	incomingWarpPrefix      = 0x7
	outgoingWarpPrefix      = 0x8
	blockPrefix             = 0x9
	feeMarketPrefix         = 0xa
	EpochExitRegistryPrefix = 0xf2
	EpochExitPrefix         = 0xf3
	ArcadiaBidPrefix        = 0xf4
)

const (
	// ToDO: clean up
	BalanceChunks   uint16 = 1
	AssetChunks     uint16 = 5
	OrderChunks     uint16 = 2
	LoanChunks      uint16 = 1
	EpochExitChunks uint16 = 1
)

var (
	failureByte  = byte(0x0)
	successByte  = byte(0x1)
	heightKey    = []byte{heightPrefix}
	timestampKey = []byte{timestampPrefix}
	feeKey       = []byte{feePrefix}
	feeMarketKey = []byte{feeMarketPrefix}

	balanceKeyPool = sync.Pool{
		New: func() any {
			return make([]byte, 1+codec.AddressLen+consts.Uint16Len)
		},
	}
)

// [txPrefix] + [txID]
func TxKey(id ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen)
	k[0] = txPrefix
	copy(k[1:], id[:])
	return
}

func StoreTransaction(
	_ context.Context,
	db database.KeyValueWriter,
	id ids.ID,
	t int64,
	success bool,
	units fees.Dimensions,
	fee uint64,
) error {
	k := TxKey(id)
	v := make([]byte, consts.Uint64Len+1+fees.DimensionsLen+consts.Uint64Len)
	binary.BigEndian.PutUint64(v, uint64(t))
	if success {
		v[consts.Uint64Len] = successByte
	} else {
		v[consts.Uint64Len] = failureByte
	}
	copy(v[consts.Uint64Len+1:], units.Bytes())
	binary.BigEndian.PutUint64(v[consts.Uint64Len+1+fees.DimensionsLen:], fee)
	return db.Put(k, v)
}

func GetTransaction(
	_ context.Context,
	db database.KeyValueReader,
	id ids.ID,
) (bool, int64, bool, fees.Dimensions, uint64, error) {
	k := TxKey(id)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return false, 0, false, fees.Dimensions{}, 0, nil
	}
	if err != nil {
		return false, 0, false, fees.Dimensions{}, 0, err
	}
	t := int64(binary.BigEndian.Uint64(v))
	success := true
	if v[consts.Uint64Len] == failureByte {
		success = false
	}
	d, err := fees.UnpackDimensions(v[consts.Uint64Len+1 : consts.Uint64Len+1+fees.DimensionsLen])
	if err != nil {
		return false, 0, false, fees.Dimensions{}, 0, err
	}
	fee := binary.BigEndian.Uint64(v[consts.Uint64Len+1+fees.DimensionsLen:])
	return true, t, success, d, fee, nil
}

// [accountPrefix] + [address]
func BalanceKey(pk codec.Address) (k []byte) {
	k = balanceKeyPool.Get().([]byte)
	k[0] = balancePrefix
	copy(k[1:], pk[:])
	binary.BigEndian.PutUint16(k[1+codec.AddressLen:], BalanceChunks)
	return
}

// If locked is 0, then account does not exist
func GetBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
) (uint64, error) {
	key, bal, _, err := getBalance(ctx, im, addr)
	balanceKeyPool.Put(key)
	return bal, err
}

func getBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
) ([]byte, uint64, bool, error) {
	k := BalanceKey(addr)
	bal, exists, err := innerGetBalance(im.GetValue(ctx, k))
	return k, bal, exists, err
}

// Used to serve RPC queries
func GetBalanceFromState(
	ctx context.Context,
	f ReadState,
	addr codec.Address,
) (uint64, error) {
	k := BalanceKey(addr)
	values, errs := f(ctx, [][]byte{k})
	bal, _, err := innerGetBalance(values[0], errs[0])
	balanceKeyPool.Put(k)
	return bal, err
}

func innerGetBalance(
	v []byte,
	err error,
) (uint64, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return binary.BigEndian.Uint64(v), true, nil
}

func SetBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	balance uint64,
) error {
	k := BalanceKey(addr)
	return setBalance(ctx, mu, k, balance)
}

func setBalance(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	balance uint64,
) error {
	return mu.Insert(ctx, key, binary.BigEndian.AppendUint64(nil, balance))
}

func DeleteBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
) error {
	return mu.Remove(ctx, BalanceKey(addr))
}

func AddBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	amount uint64,
	create bool,
) error {
	key, bal, exists, err := getBalance(ctx, mu, addr)
	if err != nil {
		return err
	}
	// Don't add balance if account doesn't exist. This
	// can be useful when processing fee refunds.
	if !exists && !create {
		return nil
	}
	nbal, err := smath.Add64(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add balance (bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			bal,
			codec.MustAddressBech32(tconsts.HRP, addr),
			amount,
		)
	}
	return setBalance(ctx, mu, key, nbal)
}

func SubBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	amount uint64,
) error {
	key, bal, _, err := getBalance(ctx, mu, addr)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract balance (bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			bal,
			codec.MustAddressBech32(tconsts.HRP, addr),
			amount,
		)
	}
	if nbal == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, key)
	}
	return setBalance(ctx, mu, key, nbal)
}

// [assetPrefix] + [address]
func AssetKey(asset ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen+consts.Uint16Len)
	k[0] = assetPrefix
	copy(k[1:], asset[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen:], AssetChunks)
	return
}

func EpochExitRegistryKey() []byte {
	// state key must >= 2 bytes
	k := make([]byte, 1+consts.Uint16Len)
	k[0] = EpochExitRegistryPrefix
	binary.BigEndian.PutUint16(k[1:], EpochExitChunks)
	return k
}

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
	// k := make([]byte, 1+len(namespace)+consts.Uint16Len)
	// k[0] = anchorPrefix
	// copy(k[1:], namespace[:])
	// binary.BigEndian.PutUint16(k[1+len(namespace):], BalanceChunks) //TODO: update the BalanceChunks to AnchorChunks
	// return string(k)
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

func EpochExitKey(epoch uint64) []byte {
	k := make([]byte, 1+8+consts.Uint16Len)
	k[0] = EpochExitPrefix
	binary.BigEndian.PutUint64(k[1:], epoch)
	binary.BigEndian.PutUint16(k[9:], EpochExitChunks)
	return k
}

// This should get all the exits for 1 epoch
func GetEpochExit(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) (*EpochExitInfo, error) {
	_, ep, _, err := getEpochExit(ctx, im, epoch)
	return ep, err
}

func getEpochExit(
	ctx context.Context,
	im state.Immutable,
	epoch uint64,
) ([]byte, *EpochExitInfo, bool, error) {
	k := EpochExitKey(epoch)
	epochExit, exists, err := innerGetEpochExit(im.GetValue(ctx, k))
	return k, epochExit, exists, err
}

// Used to serve RPC queries
func GetEpochExitsFromState(
	ctx context.Context,
	f ReadState,
	epoch uint64,
) (*EpochExitInfo, error) {
	k := EpochExitKey(epoch)
	values, errs := f(ctx, [][]byte{k})
	epochExit, _, err := innerGetEpochExit(values[0], errs[0])
	return epochExit, err
}

func innerGetEpochExit(
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
	info, err := UnmarshalEpochExitInfo(p)
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func SetEpochExit(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
	info *EpochExitInfo,
) error {
	k := EpochExitKey(epoch)
	return setEpochExit(ctx, mu, k, info)
}

func setEpochExit(
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

// TODO I should be able to delete 1 item out of the list instead of having to delete them all at once.
func DelEpochExit(
	ctx context.Context,
	mu state.Mutable,
	epoch uint64,
) error {
	k := EpochExitKey(epoch)
	return delEpochExit(ctx, mu, k)
}

func delEpochExit(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
) error {
	return mu.Remove(ctx, key)
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
	// @todo should we send parsed values?
	return v[8 : 8+48], v[8+48:], nil
}

func PrefixBlockKey(block ids.ID, parent ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen*2)
	k[0] = blockPrefix
	copy(k[1:], block[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen*2:], LoanChunks)
	copy(k[1+ids.IDLen:], parent[:])
	return
}

func HeightKey() (k []byte) {
	return heightKey
}

func TimestampKey() (k []byte) {
	return timestampKey
}

func FeeKey() (k []byte) {
	return feeKey
}

func FeeMarketKey() (k []byte) {
	return feeMarketKey
}
