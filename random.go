// Copyright (C) 2018. See AUTHORS.

package random

import (
	"math"
	"math/rand"
)

// paramsFromEps returns the parameters used for the random quantile estimator
// for a given epsilon.
func paramsFromEps(eps float64) (b, s int) {
	b = int(math.Ceil(-math.Log2(eps))) + 1
	s = int(math.Ceil(math.Sqrt(-math.Log2(eps)) / eps))
	return b, s
}

// blockSize returns the allocation size of the number of floats for a given
// epsilon.
func blockSize(eps float64) int {
	b, s := paramsFromEps(eps)
	return b * s
}

// EstimateEpsilon finds an epsilon within tol of the epsilon that would return
// the largest number of floats NewRandom would use under the specified floats.
func EstimateEpsilon(floats int, tol float64) float64 {
	// search for an eps such that floats ==
	//	(math.Ceil(-math.Log2(eps)) + 1) *
	//	math.Ceil(math.Sqrt(-math.Log2(eps))/eps)

	min, max := 1.0, 0.0 // the function is decreasing
	min_size, max_size := -1, -1
	for {
		guess := (min + max) / 2
		guess_size := blockSize(guess)
		switch {
		case guess_size == floats:
			// how lucky
			return guess
		case guess_size < floats:
			// we guessed too low, so set the new minimum to our guess.
			min, min_size = guess, guess_size
		case guess_size > floats:
			// we guessed too high, so set the new maximum to our guess
			max, max_size = guess, guess_size
		}
		// if there's no way for us to make progress or we're within the given
		// tolerance, just bail now and return the smaller min value, because
		// min is guaranteed to cause a size less than floats as it only gets
		// set when that is the case.
		if min_size == max_size || min-max < tol {
			return min
		}
	}
}

// Random implements the random quantile estimator. The expected usage is to
// create one, Add the points as desired, and then call Finish and never use
// the Random again. It would be unsafe to do anything else.
type Random struct {
	e float64 // epsilon
	b int     // -log(e) + 1
	s int     // sqrt(-log(e)) / e

	buffers []Buffer
	merger  *bufferMerger
	cur     *Buffer // the current buffer we're filling in

	// these values keep track of how many elements we've observed in the
	// current buffer. we only add one element per 1 << level observations.
	// so the reference impl just picks the value that lands when r.count
	// is 1 << level. I want to avoid that because there might be dependencies
	// in the ordering of the data. for example, if we had two servers reporting
	// the values that pass might be in the order of A B A B, and so we'd always
	// sample values from A or B. it is chosen's job to avoid that problem.
	count     int     // the number of elements we still need to observe
	chosen    int     // prechoose the value since we know the level
	pcg       pcg     // used to reservoir sample
	reservoir float64 // the current element in the reservoir

	// level contains what level we're currently filling. it gets set when
	// we choose a new bucket to fill based on the current value of n.
	level uint32
	// next contains when the next bump to level should occur. when n is equal
	// to next, we increment level and all the associated values depending
	// on level.
	next int64
	n    int64
}

// NewRandom calls NewRandomWithSeed with a random seed from math/rand.
func NewRandom(eps float64) *Random {
	return NewRandomWithSeed(eps, uint64(rand.Int63()))
}

// NewRandomWithSeed constructs a Random with the given epsilon tolerance for
// changes in the CDF. The seed parameter lets one choose what seed to use for
// the collection of the stream.
func NewRandomWithSeed(eps float64, seed uint64) *Random {
	b, s := paramsFromEps(eps)

	// allocate all the space for the buffers in one allocation and dole them
	// out to each buffer
	block := make([]float64, b*s)

	buffers := make([]Buffer, b)
	for i := range buffers {
		start := int(s) * i
		end := start + int(s)
		buffers[i] = newBuffer(block[start:end:end])
	}
	buf := &buffers[0]
	buf.Level = 0

	return &Random{
		e: eps,
		b: b,
		s: s,

		buffers: buffers,
		merger:  newBufferMerger(make([]float64, s), newPCG(seed, 0)),
		cur:     buf,

		count:  0,
		chosen: 1,
		pcg:    newPCG(seed, 1),

		next: int64(s) * 1 << uint(b-1),
	}
}

// resetCount resets the counter of observed values for this bucket entry to
// zero and picks the index that we'll pick for the next value.
func (r *Random) resetCount() {
	r.count = 0
	r.chosen = r.pcg.Intn(1<<r.level) + 1
}

// Add puts the value in the quantile estimator.
func (r *Random) Add(value float64) {
	// increment our counters
	r.n++
	r.count++

	// check if we should keep this value in the reservoir
	if r.count == r.chosen {
		r.reservoir = value
	}

	// check if we're ready to add this value to the buffer
	if r.count < 1<<r.level {
		return
	}

	// we are so add the value into the buffer
	r.cur.Data = append(r.cur.Data, r.reservoir)

	// if we still have room, nothing left to do besides pick what the next
	// value we'll store in the buffer is.
	if len(r.cur.Data) < cap(r.cur.Data) {
		r.resetCount()
		return
	}

	// it's full. sort it and find another buffer to fill possibly merging
	// other buffers if required.
	r.cur.sort()

	// since we filled a buffer, check if we should bump to the next level and
	// if so reset all the values to whatever the current level now is.
	if r.n == r.next {
		r.next <<= 1
		r.level++
	}
	r.resetCount()

	// first look for an empty one
	for i := range r.buffers {
		buf := &r.buffers[i]
		if buf.Level == -1 {
			r.cur = buf
			r.cur.Level = int32(r.level)
			return
		}
	}

	// shoot we didn't get lucky.
	// find the lowest level with two buffers so we can merge them. there's a
	// trivial algorithm to do this in O(N^2) time, so lets do that and
	// optimize later. this doesn't show up on benchmarking.
	//
	// TODO(jeff): investigate distribution of add latencies.

	// keep track of what the level has to be above during our search.
	above := int32(-1)
	for {
		// find the minimum level above our cutoff
		min_level := int32(-1)
		for i := range r.buffers {
			level := r.buffers[i].Level
			if (min_level == -1 || level < min_level) && level > above {
				min_level = level
			}
		}

		// this should never happen.
		if min_level == -1 {
			panic("ran out of options to merge")
		}

		// search for the first two buckets with min_level
		var b *Buffer
		for i := range r.buffers {
			buf := &r.buffers[i]
			if buf.Level != min_level {
				continue
			}

			// found the first one. store it in b and try to find another.
			if b == nil {
				b = buf
				continue
			}

			// yay we found two buckets. make the destination of the merge
			// the later one in the list so that the first one is cleared
			// found faster, and used frequently so that it stays in the
			// cache.
			r.merger.merge(buf, b)

			// merge guarantees the source is cleared, so just use it.
			r.cur = b
			r.cur.Level = int32(r.level)
			return
		}

		// nope couldn't find with that level, try again with a higher one.
		above = min_level
	}
}

// Summarize is a helper that returns a Summary for a Random.
func (r *Random) Summarize() Summary {
	return r.Finish().Summarize()
}

// FinishedRandom represents a full collection of a Random value.
type FinishedRandom struct {
	E       float64
	N       int64
	Buffers []Buffer
}

// Finish returns a FinishedRandom that can be merged and summarized. It is
// unsafe to call Add on Random after Finish has been called.
func (r *Random) Finish() FinishedRandom {
	return FinishedRandom{
		E:       r.e,
		N:       r.n,
		Buffers: r.buffers,
	}
}
