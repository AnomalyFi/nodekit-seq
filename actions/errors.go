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
	ErrNameSpaceNotRegistered          = errors.New("namespace not registered")
	ErrNotAuthorized                   = errors.New("not authorized")
	ErrNameSpaceAlreadyRegistered      = errors.New("namespace already registered")
	ErrExitEpochSmallerThanStartEpoch  = errors.New("exit epoch is smaller than start epoch")
)
