package wasmruntime

import "errors"

var (
	ErrContractCompile = errors.New("error compiling contract")
	ErrInvalidFuncSig  = errors.New("invalid function signature")
	ErrExecutionRevert = errors.New("error: execution reverted by contract")
)
