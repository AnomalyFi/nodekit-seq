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

type FIFOMap[K comparable, V any] struct {
	internalMap *ShardedMap[K, V]
	queue       chan K
	maxSize     int
	currentSize int
}

// NewFIFOMap returns a FIFO map that internally uses the sharded map. It keeps count of all the items
// inserted, and when we exceed the size, values will be evicted using the FIFO (first in, first out) policy.
func NewFIFOMap[K comparable, V any](size, shards int, hashFn HashFn[K]) *FIFOMap[K, V] {
	return &FIFOMap[K, V]{
		internalMap: NewShardedMap[K, V](size, shards, hashFn),
		queue:       make(chan K, size*2),
		maxSize:     size,
		currentSize: 0,
	}
}

func (m *FIFOMap[K, V]) Get(key K) (V, bool) {
	return m.internalMap.Get(key)
}

func (m *FIFOMap[K, V]) Put(key K, val V) {
	if !m.internalMap.Has(key) {
		// If we're about to exceed max size, remove first value from the map
		if m.currentSize >= m.maxSize {
			f := <-m.queue
			m.internalMap.Del(f)
			m.currentSize--
		}

		m.internalMap.Put(key, val)
		m.queue <- key
		m.currentSize++
	} else {
		m.internalMap.Put(key, val)
	}
}

func (m *FIFOMap[K, V]) Has(key K) bool {
	return m.internalMap.Has(key)
}

func (m *FIFOMap[K, V]) Del(key K) {
	if m.internalMap.Has(key) {
		m.internalMap.Del(key)
		m.currentSize--
	}
}

func (m *FIFOMap[K, V]) Len() int {
	return m.currentSize
}
