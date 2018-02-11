// Copyright (C) 2018. See AUTHORS.

package random

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
