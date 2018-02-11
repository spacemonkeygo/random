// Copyright (C) 2018. See AUTHORS.

package random

// bufferMerger is a thing that can merge two buffers
type bufferMerger struct {
	coin    coin
	scratch []float64
}

// newBufferMerger creates a buffer merger with the associated scratch space
func newBufferMerger(scratch []float64, pcg pcg) *bufferMerger {
	return &bufferMerger{
		scratch: scratch,
		coin: coin{
			pcg: pcg,
		},
	}
}

// merge takes half of the values in b and puts them into a. the slices should
// be sorted.
func (b *bufferMerger) merge(dst, other *Buffer) {
	b.scratch = b.scratch[:0]
	merge := newMergeSorter([]mergeItem{
		{data: dst.Data},
		{data: other.Data},
	})
	use := b.coin.toss()
	for {
		value, _, ok := merge.next()
		if !ok {
			break
		}
		if use {
			b.scratch = append(b.scratch, value)
		}
		use = !use
	}

	// increment the level and clear the other one.
	dst.Level++
	dst.Sorted = true
	copy(dst.Data, b.scratch)
	other.clear()
}
