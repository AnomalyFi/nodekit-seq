package wasm

import (
	"unsafe"
)

type TxContext struct {
	timestamp    int64
	msgSenderPtr uint32
}

func txContextToBytes(c TxContext) []byte {
	// creates array of length 2^10 and access the memory at struct c to have enough space for all the struct.
	// [:size:size] slices array to size and fixes array size as size
	size := unsafe.Sizeof(c)
	bytes := (*[1 << 10]byte)(unsafe.Pointer(&c))[:size:size]
	return bytes
}
