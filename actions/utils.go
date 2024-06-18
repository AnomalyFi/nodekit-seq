package actions

import "github.com/AnomalyFi/hypersdk/codec"

func ContainsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}
