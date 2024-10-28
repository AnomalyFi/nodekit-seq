// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import "errors"

var (
	ErrOutputValueZero                 = errors.New("value is zero")
	ErrNotWhiteListed                  = errors.New("not whitelisted")
	ErrInvalidBidderSignature          = errors.New("invalid bidder signature")
	ErrParsedBuilderSEQAddressMismatch = errors.New("parsed builder SEQ address mismatch")
	ErrOutputMemoTooLarge              = errors.New("memo is too large")
)
