// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/fees"
	tconsts "github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
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

	// stateDB
	balancePrefix             = 0x0
	assetPrefix               = 0x1
	orderPrefix               = 0x2
	loanPrefix                = 0x3
	heightPrefix              = 0x4
	timestampPrefix           = 0x5
	feePrefix                 = 0x6
	incomingWarpPrefix        = 0x7
	outgoingWarpPrefix        = 0x8
	blockPrefix               = 0x9
	relayerGasPrefix          = 0x10
	relayerGasTimeStampPrefix = 0x11
	relayerBalancePrefix      = 0x12
	feeMarketPrefix           = 0x13
)

const (
	BalanceChunks             uint16 = 1
	AssetChunks               uint16 = 5
	OrderChunks               uint16 = 2
	LoanChunks                uint16 = 1
	RelayerGasChunks          uint16 = 1
	RelayerGasTimeStampChunks uint16 = 1
)

var (
	failureByte    = byte(0x0)
	successByte    = byte(0x1)
	heightKey      = []byte{heightPrefix}
	timestampKey   = []byte{timestampPrefix}
	feeKey         = []byte{feePrefix}
	feeMarketKey   = []byte{feeMarketPrefix}
	balanceKeyPool = sync.Pool{
		New: func() any {
			return make([]byte, 1+codec.AddressLen+ids.IDLen+consts.Uint16Len)
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

// [accountPrefix] + [address] + [asset]
func BalanceKey(pk codec.Address, asset ids.ID) (k []byte) {
	k = balanceKeyPool.Get().([]byte)
	k[0] = balancePrefix
	copy(k[1:], pk[:])
	copy(k[1+codec.AddressLen:], asset[:])
	binary.BigEndian.PutUint16(k[1+codec.AddressLen+ids.IDLen:], BalanceChunks)
	return
}

// If locked is 0, then account does not exist
func GetBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	key, bal, _, err := getBalance(ctx, im, addr, asset)
	balanceKeyPool.Put(key)
	return bal, err
}

func getBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) ([]byte, uint64, bool, error) {
	k := BalanceKey(addr, asset)
	bal, exists, err := innerGetBalance(im.GetValue(ctx, k))
	return k, bal, exists, err
}

// Used to serve RPC queries
func GetBalanceFromState(
	ctx context.Context,
	f ReadState,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	k := BalanceKey(addr, asset)
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
	asset ids.ID,
	balance uint64,
) error {
	k := BalanceKey(addr, asset)
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
	asset ids.ID,
) error {
	return mu.Remove(ctx, BalanceKey(addr, asset))
}

func AddBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	amount uint64,
	create bool,
) error {
	key, bal, exists, err := getBalance(ctx, mu, addr, asset)
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
			"%w: could not add balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
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
	asset ids.ID,
	amount uint64,
) error {
	key, bal, _, err := getBalance(ctx, mu, addr, asset)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
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

// Used to serve RPC queries
func GetAssetFromState(
	ctx context.Context,
	f ReadState,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, codec.Address, error) {
	values, errs := f(ctx, [][]byte{AssetKey(asset)})
	return innerGetAsset(values[0], errs[0])
}

func GetAsset(
	ctx context.Context,
	im state.Immutable,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, codec.Address, error) {
	k := AssetKey(asset)
	return innerGetAsset(im.GetValue(ctx, k))
}

func innerGetAsset(
	v []byte,
	err error,
) (bool, []byte, uint8, []byte, uint64, codec.Address, error) {
	if errors.Is(err, database.ErrNotFound) {
		return false, nil, 0, nil, 0, codec.EmptyAddress, nil
	}
	if err != nil {
		return false, nil, 0, nil, 0, codec.EmptyAddress, err
	}
	symbolLen := binary.BigEndian.Uint16(v)
	symbol := v[consts.Uint16Len : consts.Uint16Len+symbolLen]
	decimals := v[consts.Uint16Len+symbolLen]
	metadataLen := binary.BigEndian.Uint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:])
	metadata := v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len : consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen]
	supply := binary.BigEndian.Uint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:])
	var addr codec.Address
	copy(addr[:], v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:])
	return true, symbol, decimals, metadata, supply, addr, nil
}

func SetAsset(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	symbol []byte,
	decimals uint8,
	metadata []byte,
	supply uint64,
	owner codec.Address,
) error {
	k := AssetKey(asset)
	symbolLen := len(symbol)
	metadataLen := len(metadata)
	v := make([]byte, consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+codec.AddressLen+1)
	binary.BigEndian.PutUint16(v, uint16(symbolLen))
	copy(v[consts.Uint16Len:], symbol)
	v[consts.Uint16Len+symbolLen] = decimals
	binary.BigEndian.PutUint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:], uint16(metadataLen))
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len:], metadata)
	binary.BigEndian.PutUint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:], supply)
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:], owner[:])
	return mu.Insert(ctx, k, v)
}

func DeleteAsset(ctx context.Context, mu state.Mutable, asset ids.ID) error {
	k := AssetKey(asset)
	return mu.Remove(ctx, k)
}

// [orderPrefix] + [txID]
func OrderKey(txID ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen+consts.Uint16Len)
	k[0] = orderPrefix
	copy(k[1:], txID[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen:], OrderChunks)
	return
}

func SetOrder(
	ctx context.Context,
	mu state.Mutable,
	txID ids.ID,
	in ids.ID,
	inTick uint64,
	out ids.ID,
	outTick uint64,
	supply uint64,
	owner ed25519.PublicKey,
) error {
	k := OrderKey(txID)
	v := make([]byte, ids.IDLen*2+consts.Uint64Len*3+ed25519.PublicKeyLen)
	copy(v, in[:])
	binary.BigEndian.PutUint64(v[ids.IDLen:], inTick)
	copy(v[ids.IDLen+consts.Uint64Len:], out[:])
	binary.BigEndian.PutUint64(v[ids.IDLen*2+consts.Uint64Len:], outTick)
	binary.BigEndian.PutUint64(v[ids.IDLen*2+consts.Uint64Len*2:], supply)
	copy(v[ids.IDLen*2+consts.Uint64Len*3:], owner[:])
	return mu.Insert(ctx, k, v)
}

func GetOrder(
	ctx context.Context,
	im state.Immutable,
	order ids.ID,
) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	ed25519.PublicKey, // owner
	error,
) {
	k := OrderKey(order)
	v, err := im.GetValue(ctx, k)
	return innerGetOrder(v, err)
}

// Used to serve RPC queries
func GetOrderFromState(
	ctx context.Context,
	f ReadState,
	order ids.ID,
) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	ed25519.PublicKey, // owner
	error,
) {
	values, errs := f(ctx, [][]byte{OrderKey(order)})
	return innerGetOrder(values[0], errs[0])
}

func innerGetOrder(v []byte, err error) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	ed25519.PublicKey, // owner
	error,
) {
	if errors.Is(err, database.ErrNotFound) {
		return false, ids.Empty, 0, ids.Empty, 0, 0, ed25519.EmptyPublicKey, nil
	}
	if err != nil {
		return false, ids.Empty, 0, ids.Empty, 0, 0, ed25519.EmptyPublicKey, err
	}
	var in ids.ID
	copy(in[:], v[:ids.IDLen])
	inTick := binary.BigEndian.Uint64(v[ids.IDLen:])
	var out ids.ID
	copy(out[:], v[ids.IDLen+consts.Uint64Len:ids.IDLen*2+consts.Uint64Len])
	outTick := binary.BigEndian.Uint64(v[ids.IDLen*2+consts.Uint64Len:])
	supply := binary.BigEndian.Uint64(v[ids.IDLen*2+consts.Uint64Len*2:])
	var owner ed25519.PublicKey
	copy(owner[:], v[ids.IDLen*2+consts.Uint64Len*3:])
	return true, in, inTick, out, outTick, supply, owner, nil
}

func DeleteOrder(ctx context.Context, mu state.Mutable, order ids.ID) error {
	k := OrderKey(order)
	return mu.Remove(ctx, k)
}

// [loanPrefix] + [asset] + [destination]
func LoanKey(asset ids.ID, destination ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen*2+consts.Uint16Len)
	k[0] = loanPrefix
	copy(k[1:], asset[:])
	copy(k[1+ids.IDLen:], destination[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen*2:], LoanChunks)
	return
}

func PrefixBlockKey(block ids.ID, parent ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen*2)
	k[0] = blockPrefix
	copy(k[1:], block[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen*2:], LoanChunks)
	copy(k[1+ids.IDLen:], parent[:])
	return
}

func RelayerGasPriceKey(relayerID uint32) (k []byte) {
	k = make([]byte, 1+consts.Uint32Len+consts.Uint16Len)
	k[0] = relayerGasPrefix
	binary.BigEndian.PutUint32(k[1:], relayerID)
	binary.BigEndian.PutUint16(k[1+consts.IntLen:], RelayerGasChunks)
	return
}

// Relayer Gas Price is the price per byte in SEQ reported by oracle for posting data to the DA layer.
func StoreRelayerGasPrice(
	ctx context.Context,
	mu state.Mutable,
	relayerID uint32,
	price uint64,
) error {
	k := RelayerGasPriceKey(relayerID)
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, price))
}

func GetRelayerGasPrice(
	ctx context.Context,
	im state.Immutable,
	relayerID uint32,
) (uint64, error) {
	k := RelayerGasPriceKey(relayerID)
	v, err := im.GetValue(ctx, k)
	if errors.Is(err, database.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(v), nil
}

func RelayerGasPriceUpdateTimeStampKey(relayerID uint32) (k []byte) {
	k = make([]byte, 1+consts.Uint32Len+consts.Uint16Len)
	k[0] = relayerGasTimeStampPrefix
	binary.BigEndian.PutUint32(k[1:], relayerID)
	binary.BigEndian.PutUint16(k[1+consts.IntLen:], RelayerGasTimeStampChunks)
	return
}

func StoreRelayerGasPriceUpdateTimeStamp(
	ctx context.Context,
	mu state.Mutable,
	relayerID uint32,
	timeStamp int64,
) error {
	k := RelayerGasPriceUpdateTimeStampKey(relayerID)
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, uint64(timeStamp)))
}

func GetRelayerGasPriceUpdateTimeStamp(
	ctx context.Context,
	im state.Immutable,
	relayerID uint32,
) (int64, error) {
	k := RelayerGasPriceUpdateTimeStampKey(relayerID)
	v, err := im.GetValue(ctx, k)
	if errors.Is(err, database.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(v)), nil
}

func RelayerBalanceKey(relayerID uint32) (k []byte) {
	k = make([]byte, 1+consts.Uint32Len+consts.Uint16Len)
	k[0] = relayerBalancePrefix
	binary.BigEndian.PutUint32(k[1:], relayerID)
	binary.BigEndian.PutUint16(k[1+consts.Uint32Len:], RelayerGasChunks)
	return k
}

func AddRelayerBalance(
	ctx context.Context,
	mu state.Mutable,
	relayerID uint32,
	amount uint64,
) error {
	k := RelayerBalanceKey(relayerID)
	bal, err := GetRelayerBalance(ctx, mu, relayerID)
	if err != nil {
		return err
	}
	nbal, err := smath.Add64(bal, amount)
	if err != nil {
		return err
	}
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, nbal))
}

func SubRelayerBalance(
	ctx context.Context,
	mu state.Mutable,
	relayerID uint32,
	amount uint64,
) error {
	k := RelayerBalanceKey(relayerID)
	bal, err := GetRelayerBalance(ctx, mu, relayerID)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return err
	}
	if nbal == 0 {
		return mu.Remove(ctx, k)
	}
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, nbal))
}

func GetRelayerBalance(
	ctx context.Context,
	im state.Immutable,
	relayerID uint32,
) (uint64, error) {
	k := RelayerBalanceKey(relayerID)
	val, _, err := innerGetBalance(im.GetValue(ctx, k))
	if err != nil {
		return 0, err
	}
	return val, nil
}

func GetRelayerBalanceFromState(
	ctx context.Context,
	f ReadState,
	relayerID uint32,
) (uint64, error) {
	k := RelayerBalanceKey(relayerID)
	values, errs := f(ctx, [][]byte{k})
	bal, _, err := innerGetBalance(values[0], errs[0])
	return bal, err
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
