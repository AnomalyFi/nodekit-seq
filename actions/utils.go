package actions

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
)

func IsWhiteListed(rules chain.Rules, actor codec.Address) bool {
	whitelistedAddressesB, _ := rules.FetchCustom("whitelisted.Addresses")
	whitelistedAddresses := whitelistedAddressesB.([]codec.Address)
	return ContainsAddress(whitelistedAddresses, actor)
}

func ContainsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

func ArcadiaFundAddress() codec.Address {
	b := make([]byte, 33)
	b[32] = 0x1
	addr, _ := codec.ToAddress(b)
	return addr
}

func MarshalAuctionInfo(p *codec.Packer, info *AuctionInfo) {
	p.PackUint64(info.EpochNumber)
	p.PackUint64(info.BidPrice)
	p.PackAddress(info.BuilderSEQAddress)
}

func UnmarshalAnchorInfo(p *codec.Packer) (*AuctionInfo, error) {
	var auctionInfo AuctionInfo
	auctionInfo.EpochNumber = p.UnpackUint64(true)
	auctionInfo.BidPrice = p.UnpackUint64(true)
	p.UnpackAddress(&auctionInfo.BuilderSEQAddress)
	if p.Err() != nil {
		return nil, p.Err()
	}
	return &auctionInfo, nil
}

func Epoch(blockHeight uint64, epochLength int64) uint64 {
	return uint64(int64(blockHeight) / epochLength)
}
