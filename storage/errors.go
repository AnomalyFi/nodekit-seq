// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import "errors"

var (
	ErrInvalidBalance = errors.New("invalid balance")
	ErrCertExists     = errors.New("cert exists for chunk layer")
)
