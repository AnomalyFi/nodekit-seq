// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/AnomalyFi/hypersdk/codec"

	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/ava-labs/avalanchego/ids"
)

type ReadState func(context.Context, [][]byte) ([][]byte, []error)

const (
	// metaDB
	txPrefix = 0x0

	// TODO: clean up the prefixes below
	// stateDB
	balancePrefix   = 0x0
	heightPrefix    = 0x4
	timestampPrefix = 0x5
	feePrefix       = 0x6
	blockPrefix     = 0x9
	feeMarketPrefix = 0xa

	ArcadiaRegistryPrefix      = 0xf2
	EpochExitRegistryPrefix    = 0xf3
	EpochExitPrefix            = 0xf4
	ArcadiaBidPrefix           = 0xf5
	GlobalRollupRegistryPrefix = 0xf6
	ArcadiaInfoPrefix          = 0xf7
)

const (
	// ToDO: clean up
	BalanceChunks   uint16 = 1
	AssetChunks     uint16 = 5
	EpochExitChunks uint16 = 1
	// The length of data stored at GlobalRollupRegistryKey and ArcadiaRegistryKey depends on number of rollups registered.
	// Each chunk gives a state storage of 64 bytes.
	// Its safe to limit the data of state storage for GlobalRollupRegistryKey and ArcadiaRegistryKey to atleast 3 KiB.
	GlobalRollupRegistryChunks uint16 = 3 * 16
	ArcadiaRegistryChunks      uint16 = 3 * 16
	// 2 AddressLen* 33 + 1 MaxNameSpaceLen *  32 = 98 bytes
	ArcadiaInfoChunks uint16 = 2
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

func PrefixBlockKey(block ids.ID, parent ids.ID) (k []byte) {
	k = make([]byte, 1+ids.IDLen*2)
	k[0] = blockPrefix
	copy(k[1:], block[:])
	binary.BigEndian.PutUint16(k[1+ids.IDLen*2:], BalanceChunks)
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
