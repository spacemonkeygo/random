// Copyright (C) 2018. See AUTHORS.

package random

import (
	"math"
	"math/rand"
	"testing"
)

func TestMonotonic_Normal(t *testing.T) {
	// TODO(jeff): i don't really know how to test that this works. the low
	// values are critical for testing weird edge conditions but don't have
	// nice statistical properties that we can assert.
	// so we'll just make sure it's increasing for each larger percentile.
	for _, eps := range []float64{0.5, 0.1, 0.05, 0.01, 0.001, 0.0001, 0.00001} {
		t.Logf("eps:%v", eps)

		r := NewRandom(eps)
		Seed(r, rand.NormFloat64)
		s := r.Summarize()

		last := s.Query(0.00)
		for ptile := 0.0; ptile <= 1.0; ptile += 1.0 / 64 {
			qu, ex := s.Query(ptile), probit(ptile)
			t.Logf("%0.2f,%v,%v,%v", ptile, qu, ex, math.Abs(qu-ex))
			if query := s.Query(ptile); query < last {
				t.Fatalf("%v < %v", query, last)
			}
		}
		t.Logf("err:%v", L1Norm(s, probit))
	}
}

func TestNormDecreases_Normal(t *testing.T) {
	r1 := NewRandom(0.5)
	Seed(r1, rand.NormFloat64)
	n1 := L1Norm(r1.Summarize(), probit)

next:
	// we don't test very small epsilons because we have to add a crazy amount
	// of data to get that level of accuracy for the test to make sense.
	for _, eps := range []float64{0.1, 0.05, 0.01, 0.001} {
		t.Logf("eps:%v", eps)

		for i := 0; i < 50; i++ {
			r2 := NewRandom(eps)
			Seed(r2, rand.NormFloat64)
			n2 := L1Norm(r2.Summarize(), probit)
			if n2 < n1 {
				t.Logf("took:%d tries", i)
				n1 = n2
				continue next
			}
		}

		t.Fatalf("failed to reduce error")
	}
}

func TestEstimateEpsilon(t *testing.T) {
	for i := 0; i < 1000; i++ {
		ask := rand.Intn(10000) + 5
		eps := EstimateEpsilon(ask, 0.0000001)
		size := blockSize(eps)
		t.Logf("ask:%d got:%d", ask, size)
		if size > ask {
			t.Fatalf("%d > %d", size, ask)
		}
	}
}

//
// benchmarks
//

func benchmarkAdd(b *testing.B, cons func() float64, eps float64) {
	val := cons()
	r := NewRandom(eps)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r.Add(val)
	}
}

func BenchmarkAddNormal_5(b *testing.B) {
	benchmarkAdd(b, rand.NormFloat64, 0.5)
}

func BenchmarkAddNormal_05(b *testing.B) {
	benchmarkAdd(b, rand.NormFloat64, 0.05)
}

func BenchmarkAddNormal_01(b *testing.B) {
	benchmarkAdd(b, rand.NormFloat64, 0.01)
}

func BenchmarkAddNormal_001(b *testing.B) {
	benchmarkAdd(b, rand.NormFloat64, 0.001)
}

func BenchmarkAddNormal_0001(b *testing.B) {
	benchmarkAdd(b, rand.NormFloat64, 0.0001)
}

func benchmarkSummarize(b *testing.B, cons func() float64, eps float64) {
	r := NewRandom(eps)
	for i := 0; i < 100000; i++ {
		r.Add(cons())
	}
	eles := make([]summaryElement, 0, r.b*r.s)
	f := r.Finish()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		f.summarize(eles[:0])
	}
}

func BenchmarkSummarizeNormal_5(b *testing.B) {
	benchmarkSummarize(b, rand.NormFloat64, 0.5)
}

func BenchmarkSummarizeNormal_05(b *testing.B) {
	benchmarkSummarize(b, rand.NormFloat64, 0.05)
}

func BenchmarkSummarizeNormal_01(b *testing.B) {
	benchmarkSummarize(b, rand.NormFloat64, 0.01)
}

func BenchmarkSummarizeNormal_001(b *testing.B) {
	benchmarkSummarize(b, rand.NormFloat64, 0.001)
}

func BenchmarkSummarizeNormal_0001(b *testing.B) {
	benchmarkSummarize(b, rand.NormFloat64, 0.0001)
}
