package types

// MIT License

// Copyright (c) 2022 Chainbound

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

type HashFn[K comparable] func(k K) uint64

func HashBytes(b []byte) uint64 {
	var hash uint64 = 5381 // magic constant, apparently this hash fewest collisions possible.

	for _, chr := range b {
		hash = ((hash << 5) + hash) + uint64(chr)
	}
	return hash
}

func HashString(s string) uint64 {
	var hash uint64 = 5381 // magic constant, apparently this hash fewest collisions possible.

	for _, chr := range s {
		hash = ((hash << 5) + hash) + uint64(chr)
	}
	return hash
}

func HashUint32(u uint32) uint64 {
	return HashUint64(uint64(u))
}

func HashUint16(u uint16) uint64 {
	return HashUint64(uint64(u))
}

func HashUint8(u uint8) uint64 {
	return HashUint64(uint64(u))
}

func HashInt64(i int64) uint64 {
	return HashUint64(uint64(i))
}

func HashInt32(i int32) uint64 {
	return HashUint64(uint64(i))
}

func HashInt16(i int16) uint64 {
	return HashUint64(uint64(i))
}

func HashInt8(i int8) uint64 {
	return HashUint64(uint64(i))
}

func HashInt(i int) uint64 {
	return HashUint64(uint64(i))
}

func HashUint(i uint) uint64 {
	return HashUint64(uint64(i))
}

func HashUint64(u uint64) uint64 {
	u ^= u >> 33
	u *= 0xff51afd7ed558ccd
	u ^= u >> 33
	u *= 0xc4ceb9fe1a85ec53
	u ^= u >> 33
	return u
}
