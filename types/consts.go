package types

import "errors"

var (
	ErrCertChunkIDNotExists = errors.New("cert chunk id not exists")
	ErrCertNotExists        = errors.New("cert not exists")
)
