// Copyright (C) 2018. See AUTHORS.

package random

import (
	"math"
	"sort"
)

// summaryElement is a list of elements for a summary for fast queries.
type summaryElement struct {
	rank  int64
	value float64
}

// Summary is produced by a Random and can answer queries about the
// distribution that was observed.
type Summary struct {
	n        float64
	elements []summaryElement
}

// numElements returns the number of elements stored in the finished Random.
func (r FinishedRandom) numElements() int {
	if len(r.Buffers) == 0 {
		return 0
	}
	return len(r.Buffers) * len(r.Buffers[0].Data)
}

// Summarize creates a Summary for querying.
func (r FinishedRandom) Summarize() Summary {
	// factor out the allocation for benchmarking.
	return r.summarize(make([]summaryElement, 0, r.numElements()))
}

func (r FinishedRandom) summarize(elements []summaryElement) Summary {
	// we summarize in a two step process. step one is to create a slice of
	// summary elements with the rank actually being the level. step two is
	// to create a rolling sum of the levels and fix up the ranks.

	// make the slices that we're going to sort with their associated levels.
	items := make([]mergeItem, 0, len(r.Buffers))
	for i := range r.Buffers {
		buf := &r.Buffers[i]

		// if the buffer has no data, there's no point in considering it for
		// merging
		if len(buf.Data) == 0 {
			continue
		}

		// ensure the buffer is sorted
		if !buf.Sorted {
			buf.sort()
		}

		// add the merge buffer and associate the data with the level.
		items = append(items, mergeItem{
			data:  buf.Data,
			level: int64(buf.Level),
		})
	}

	// create the list of summary elements sorted by value with the ranks
	// growing by the buffer's level.
	merge := newMergeSorter(items)
	rank := int64(0)
	for {
		value, level, ok := merge.next()
		if !ok {
			break
		}
		elements = append(elements, summaryElement{
			rank:  rank,
			value: value,
		})
		rank += (1 << uint64(level))
	}

	return Summary{
		n:        float64(r.N),
		elements: elements,
	}
}

// Query returns the estimated value at the given percentile.
func (s Summary) Query(ptile float64) float64 {
	target := int64(math.Ceil(s.n * ptile))
	idx := sort.Search(len(s.elements), func(idx int) bool {
		return s.elements[idx].rank >= target
	})
	if idx >= len(s.elements) {
		return s.elements[len(s.elements)-1].value
	}
	below_idx, above_idx := 0, len(s.elements)
	if idx < len(s.elements) {
		above_idx = idx
	}
	if idx > 1 {
		below_idx = idx - 1
	}
	if above_idx == below_idx {
		return s.elements[above_idx].value
	}
	below, above := s.elements[below_idx], s.elements[above_idx]
	x := float64(target-below.rank) / float64(above.rank-below.rank)
	return below.value + (above.value-below.value)*x
}
