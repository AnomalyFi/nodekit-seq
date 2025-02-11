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

	// stateDB
	balancePrefix        = 0x0
	heightPrefix         = 0x4
	timestampPrefix      = 0x5
	feePrefix            = 0x6
	blockPrefix          = 0x9
	feeMarketPrefix      = 0xa
	DACertToBNoncePrefix = 0xb0
	EpochToBNoncePrefix  = 0xb1
	DACertIndexPrefix    = 0xb2
	DACertPrefix         = 0xb3
	DACertChunkIDPrefix  = 0xb4
)

const (
	BalanceChunks              uint16 = 1
	DACertficateToBNonceChunks uint16 = 1
	EpochToBNonceChunks        uint16 = 1
	DACertficateIndexChunks    uint16 = 20
	DACertificateChunks        uint16 = 1
	DACertificateChunkIDChunks uint16 = 1
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
