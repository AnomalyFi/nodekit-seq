// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import "errors"

var (
	ErrNoSwapToFill                    = errors.New("no swap to fill")
	ErrRelayerIDsUnitGasPricesMismatch = errors.New("len of relayerIDs and unitGasPrices mismatched")
	ErrNotWhiteListed                  = errors.New("address not whitelisted")
)
