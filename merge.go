// Copyright (C) 2018. See AUTHORS.

package random

import (
	"fmt"
	"sort"
)

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
	merger := newBufferMerger(make([]float64, s), newPCG(seed, 0))
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
