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
	deployID      uint8 = 6
	transactID    uint8 = 7
)

const (
	// TODO: tune this
	BurnComputeUnits        = 2
	CreateAssetComputeUnits = 10
	MintAssetComputeUnits   = 2
	TransferComputeUnits    = 1

	MsgComputeUnits = 15

	OracleComputeUnits = 10

	// Max chunks
	DeployMaxChunks uint16 = 20_000

	// W.S.C Compute Units
	DeployComputeUnits   uint64 = 1_280_000
	TransactComputeUnits uint64 = 10_000
	BaseComputeUnits     uint64 = 10_000

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9
)

var DefaultNMTNamespace = make([]byte, 8)
