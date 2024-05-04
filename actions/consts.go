// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

// Note: Registry will error during initialization if a duplicate ID is assigned. We explicitly assign IDs to avoid accidental remapping.
const (
	burnAssetID   uint8 = 0
	closeOrderID  uint8 = 1
	createAssetID uint8 = 2
	exportAssetID uint8 = 3
	importAssetID uint8 = 4
	createOrderID uint8 = 5
	fillOrderID   uint8 = 6
	mintAssetID   uint8 = 7
	transferID    uint8 = 8

	DeployID   uint8 = 9
	TransactID uint8 = 10
)

const (
	// TODO: tune this
	BurnComputeUnits        = 2
	CloseOrderComputeUnits  = 5
	CreateAssetComputeUnits = 10
	ExportAssetComputeUnits = 10
	ImportAssetComputeUnits = 10
	CreateOrderComputeUnits = 5
	NoFillOrderComputeUnits = 5
	FillOrderComputeUnits   = 15
	MintAssetComputeUnits   = 2
	TransferComputeUnits    = 1

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9

	// Max chunks
	DeployMaxChunks uint16 = 20_000

	// Compute Units
	DeployMaxComputeUnits   uint64 = 1_280_000 // DeployMaxChunks x 64(size of chunk)
	TransactMaxComputeUnits uint64 = 10_000_000
	BaseComputeUnits        uint64 = 10_000
	// Num of statekeys
	NumStateKeys int = 1024 //@todo make this changable with SEQ authentication
)

var DefaultNMTNamespace = make([]byte, 8)
