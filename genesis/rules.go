// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/fees"
	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/nodekit-seq/storage"
)

var _ chain.Rules = (*Rules)(nil)

type Rules struct {
	g *Genesis

	networkID uint32
	chainID   ids.ID
}

// TODO: use upgradeBytes
func (g *Genesis) Rules(_ int64, networkID uint32, chainID ids.ID) *Rules {
	return &Rules{g, networkID, chainID}
}

func (r *Rules) NetworkID() uint32 {
	return r.networkID
}

func (r *Rules) ChainID() ids.ID {
	return r.chainID
}

func (r *Rules) GetMinBlockGap() int64 {
	return r.g.MinBlockGap
}

func (r *Rules) GetMinEmptyBlockGap() int64 {
	return r.g.MinEmptyBlockGap
}

func (r *Rules) GetValidityWindow() int64 {
	return r.g.ValidityWindow
}

func (r *Rules) GetMaxActionsPerTx() uint8 {
	return r.g.MaxActionsPerTx
}

func (r *Rules) GetMaxOutputsPerAction() uint8 {
	return r.g.MaxOutputsPerAction
}

func (r *Rules) GetMaxBlockUnits() fees.Dimensions {
	return r.g.MaxBlockUnits
}

func (r *Rules) GetBaseComputeUnits() uint64 {
	return r.g.BaseComputeUnits
}

func (*Rules) GetSponsorStateKeysMaxChunks() []uint16 {
	return []uint16{storage.BalanceChunks}
}

func (r *Rules) GetStorageKeyReadUnits() uint64 {
	return r.g.StorageKeyReadUnits
}

func (r *Rules) GetStorageValueReadUnits() uint64 {
	return r.g.StorageValueReadUnits
}

func (r *Rules) GetStorageKeyAllocateUnits() uint64 {
	return r.g.StorageKeyAllocateUnits
}

func (r *Rules) GetStorageValueAllocateUnits() uint64 {
	return r.g.StorageValueAllocateUnits
}

func (r *Rules) GetStorageKeyWriteUnits() uint64 {
	return r.g.StorageKeyWriteUnits
}

func (r *Rules) GetStorageValueWriteUnits() uint64 {
	return r.g.StorageValueWriteUnits
}

func (r *Rules) GetMinUnitPrice() fees.Dimensions {
	return r.g.MinUnitPrice
}

func (r *Rules) GetUnitPriceChangeDenominator() fees.Dimensions {
	return r.g.UnitPriceChangeDenominator
}

func (r *Rules) GetWindowTargetUnits() fees.Dimensions {
	return r.g.WindowTargetUnits
}

func (r *Rules) GetFeeMarketPriceChangeDenominator() uint64 {
	return r.g.FeeMarketPriceChangeDenominator
}

func (r *Rules) GetFeeMarketWindowTargetUnits() uint64 {
	return r.g.FeeMarketWindowTargetUnits
}

func (r *Rules) GetFeeMarketMinUnitPrice() uint64 {
	return r.g.FeeMarketMinUnits
}

func (r *Rules) FetchCustom(s string) (any, bool) {
	if s == "whitelisted.Addresses" {
		return r.g.Config.GetParsedWhiteListedAddress(), false
	}
	return nil, false
}
