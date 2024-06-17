// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

// Note: Registry will error during initialization if a duplicate ID is assigned. We explicitly assign IDs to avoid accidental remapping.
const (
	burnAssetID   uint8 = 0
	createAssetID uint8 = 1
	mintAssetID   uint8 = 2
	transferID    uint8 = 3
	msgID         uint8 = 4
	oracleID      uint8 = 5
)

const (
	// TODO: tune this
	BurnComputeUnits        = 2
	CreateAssetComputeUnits = 10
	MintAssetComputeUnits   = 2
	TransferComputeUnits    = 1

	MsgComputeUnits = 15

	OracleComputeUnits = 1000

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9
)

var DefaultNMTNamespace = make([]byte, 8)
