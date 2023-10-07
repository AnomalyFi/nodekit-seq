package types

import (
	"sync"
	"testing"
)

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

func TestMap(t *testing.T) {
	sm := NewShardedMap[int, int](1000, 10, HashInt)
	sm.Put(1, 10)

	if !sm.Has(1) {
		t.Fail()
	}

	if v, ok := sm.Get(1); ok {
		if v != 10 {
			t.Fail()
		}
	}

}

func BenchmarkSyncMapPut(b *testing.B) {
	var m sync.Map
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(i, i)
	}
}

func BenchmarkSyncMapGet(b *testing.B) {
	var m sync.Map
	for i := 0; i < b.N; i++ {
		m.Store(i, i)
	}
	b.ResetTimer()
	var v int
	for i := 0; i < b.N; i++ {
		if val, ok := m.Load(i); ok {
			v = val.(int)
		}
	}

	_ = v
}

func BenchmarkShardMapPut(b *testing.B) {
	sm := NewShardedMap[int, int](b.N, 1, HashInt)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Put(i, i)
	}
}

func BenchmarkShardMapGet(b *testing.B) {
	sm := NewShardedMap[int, int](b.N, 1, HashInt)
	for i := 0; i < b.N; i++ {
		sm.Put(i, i)
	}
	b.ResetTimer()
	var v int
	for i := 0; i < b.N; i++ {
		v, _ = sm.Get(i)
	}

	_ = v
}
