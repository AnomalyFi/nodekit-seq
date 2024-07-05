// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consts

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	HRP      = "token"
	Name     = "tokenvm"
	Symbol   = "TKN"
	Decimals = 9
)

const (
	// Num of statekeys
	NumStaticStateKeys int = 128
)

var ID ids.ID

func init() {
	b := make([]byte, ids.IDLen)
	copy(b, []byte(Name))
	vmID, err := ids.ToID(b)
	if err != nil {
		panic(err)
	}
	ID = vmID
}

// Instantiate registry here so it can be imported by any package. We set these
// values in [controller/registry].
var (
	ActionRegistry *codec.TypeParser[chain.Action, bool]
	AuthRegistry   *codec.TypeParser[chain.Auth, bool]
)
