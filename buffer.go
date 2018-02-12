// Copyright (C) 2018. See AUTHORS.

package random

import "sort"

// Buffer represents some collected data at some level. The higher the level,
// the more significant the data.
type Buffer struct {
	Data   []float64
	Level  int32
	Sorted bool
}

// newBuffer returns a new cleared buffer with the data slice as its backing
// store.
func newBuffer(data []float64) Buffer {
	return Buffer{
		Data:   data[:0],
		Level:  -1, // not full yet
		Sorted: false,
	}
}

// clear resets the buffer to the state as if it was just returned by newBuffer
func (b *Buffer) clear() {
	b.Data = b.Data[:0]
	b.Level = -1
	b.Sorted = false
}

// sort sorts the buffer's data and flags the data as sorted.
func (b *Buffer) sort() {
	sort.Float64s(b.Data)
	b.Sorted = true
}
