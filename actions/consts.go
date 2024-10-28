// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

// Note: Registry will error during initialization if a duplicate ID is assigned. We explicitly assign IDs to avoid accidental remapping.
const (
	TransferID uint8 = 0
	MsgID      uint8 = 1
	ExitID     uint8 = 2
	AuctionID  uint8 = 3
)

const (
	// TODO: tune this
	TransferComputeUnits  = 1
	EpochExitComputeUnits = 1
	AuctionComputeUnits   = 1

	MsgComputeUnits = 15

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9
)

var DefaultNMTNamespace = make([]byte, 8)
