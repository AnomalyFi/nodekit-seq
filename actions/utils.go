package actions

import (
	"encoding/binary"

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

func RelayerIDToAddress(relayerID uint32) codec.Address {
	fillBytes := make([]byte, 29)
	relayerIDBytes := binary.BigEndian.AppendUint32(nil, relayerID)
	rB := append(fillBytes, relayerIDBytes...)
	addr, _ := codec.ToAddress(rB)
	return addr
}
