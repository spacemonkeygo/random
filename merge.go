// Copyright (C) 2015 Space Monkey, Inc.

package random

import (
	"fmt"
	"sort"
)

type rng interface {
	Int63() int64
}

// coin is a simple struct to let us get random bools and make minimum calls
// to the random number generator.
type coin struct {
	rng  rng
	val  int64
	bits int
}

func (c *coin) toss() (val bool) {
	if c.bits == 0 {
		c.val = c.rng.Int63()
		c.bits = 63
	}
	c.bits--
	val = c.val&1 > 0
	c.val >>= 1
	return val
}

// mergeItem keeps track of a slice of data and what level the data is at.
// it's different than a buffer because we want to be able to mutate the
// slice, and the data is always sorted.
type mergeItem struct {
	data  []float64
	level int64
}

// mergeSorter merges the list of data slices in linear time.
type mergeSorter struct {
	items []mergeItem
}

// newMergeSorter constructs a mergeSorter from a list of buffers.
func newMergeSorter(items []mergeItem) mergeSorter {
	return mergeSorter{
		items: items,
	}
}

// next returns the minimum value from all the buffers and which buffer it
// came from. it will return false if it ran out of values.
func (m *mergeSorter) next() (val float64, level int64, ok bool) {
	if len(m.items) == 0 {
		return 0, 0, false
	}

	val, idx := m.items[0].data[0], 0
	for i := 1; i < len(m.items); i++ {
		if cand := m.items[i].data[0]; cand < val {
			val, idx = cand, i
		}
	}
	item := &m.items[idx]

	// if there's not enough values left in the item, swap the slice to the
	// last position and slice it off so we never consider it again.
	if len(item.data) <= 1 {
		last := len(m.items) - 1
		m.items[idx] = m.items[last]
		m.items = m.items[:last]
	} else {
		// otherwise, consume the entry and as many as skip requires.
		item.data = item.data[1:]
	}

	return val, item.level, true
}

// bufferMerger is a thing that can merge two buffers
type bufferMerger struct {
	coin    coin
	scratch []float64
}

// newBufferMerger creates a buffer merger with the associated scratch space
func newBufferMerger(scratch []float64, rng rng) *bufferMerger {
	return &bufferMerger{
		scratch: scratch,
		coin: coin{
			rng: rng,
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

// copyBuffers returns a deep copy of all of the buffers in the given slice.
func copyBuffers(buffers []Buffer) []Buffer {
	out := make([]Buffer, 0, len(buffers))
	for _, buf := range buffers {
		out = append(out, Buffer{
			Data:   append([]float64(nil), buf.Data...),
			Level:  buf.Level,
			Sorted: buf.Sorted,
		})
	}
	return out
}

// byLevel sorts a slice of Buffers by their level, lowest first.
type byLevel []Buffer

func (b byLevel) Len() int           { return len(b) }
func (b byLevel) Less(i, j int) bool { return b[i].Level < b[j].Level }
func (b byLevel) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// Merge will merge the specified rs into a new FinishedRandom so that it is as
// if the result observed all of the values from the passed in rs. It will
// error if any of the epsilon values are different for the FinishedRandoms.
func Merge(seed uint64, r FinishedRandom, rs ...FinishedRandom) (
	out FinishedRandom, err error) {

	// special case merging one random as the identity function
	if len(rs) == 0 {
		return r, nil
	}

	out = r
	b, s := paramsFromEps(out.E)
	buffers := make([]Buffer, 0, b*(1+len(rs)))
	rng := lcg(seed)
	merger := newBufferMerger(make([]float64, s), &rng)
	buffers = append(buffers, copyBuffers(r.Buffers)...)

	for _, r := range rs {
		if out.E != r.E {
			return out, fmt.Errorf("bad merge: e1:%v e2:%v", out.E, r.E)
		}
		out.N += r.N
		buffers = append(buffers, copyBuffers(r.Buffers)...)
	}

	for {
		merged := false
		sort.Sort(byLevel(buffers))

		for i := 0; i < len(buffers)-1; i++ {
			// attempt to merge buffers[i] and buffers[i+1]
			bl, bh := &buffers[i], &buffers[i+1]
			if bl.Level == -1 || bh.Level == -1 || bl.Level != bh.Level {
				continue
			}
			if len(bl.Data) != s || len(bh.Data) != s {
				continue
			}

			if !bl.Sorted {
				bl.sort()
			}
			if !bh.Sorted {
				bh.sort()
			}

			merged = true
			merger.merge(bh, bl)
		}

		if !merged {
			break
		}
	}

	// take the b largest buffers out and use them
	sort.Sort(byLevel(buffers))
	out.Buffers = buffers[len(buffers)-b:]

	return out, nil
}
