// Copyright (C) 2018. See AUTHORS.

package random

import (
	"math"
	"math/rand"
	"time"
)

func init() { rand.Seed(int64(time.Now().UnixNano())) }

func erfInv(val float64) float64 {
	if val == 1 {
		return math.Inf(1)
	}
	if val == -1 {
		return math.Inf(-1)
	}

	// bisect to get the value. somewhere between -100 and 100.
	min, max := -100.0, 100.0
	for {
		guess := (min + max) / 2
		guess_val := math.Erf(guess)

		if math.Abs(val-guess_val) < 0.0000001 {
			return guess
		}

		switch {
		case guess_val > val:
			max = guess
		case guess_val < val:
			min = guess
		}
	}
}

func probit(val float64) float64 {
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}
	return math.Sqrt2 * erfInv(2*val-1)
}

func L1Norm(s Summary, cdf func(float64) float64) float64 {
	const samples = 100000
	sum := 0.0
	for i := 0; i < samples; i++ {
		ptile := rand.Float64()
		approx := s.Query(ptile)
		exact := cdf(ptile)
		sum += math.Abs(approx - exact)
	}
	return sum / samples
}

func Seed(r *Random, dist func() float64) {
	for i := 0; i < 100000; i++ {
		r.Add(dist())
	}
}
