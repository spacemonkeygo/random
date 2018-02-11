// Copyright (C) 2018. See AUTHORS.

package random

import (
	"math/rand"
	"testing"
)

func TestMerge_Normal(t *testing.T) {
	const count = 10
	const eps = 0.05

	t.Logf("eps:%v ========================", eps)

	r := NewRandom(eps)
	for i := 0; i < count; i++ {
		Seed(r, rand.NormFloat64)
	}
	s_tot := r.Summarize()

	rs := make([]FinishedRandom, 0)
	for i := 0; i < count; i++ {
		r := NewRandom(eps)
		Seed(r, rand.NormFloat64)
		rs = append(rs, r.Finish())
	}
	r_mer, err := Merge(uint64(rand.Int63()), rs[0], rs[1:]...)
	if err != nil {
		t.Fatal(err)
	}
	s_mer := r_mer.Summarize()

	for ptile := 0.00; ptile < 1.01; ptile += 0.01 {
		q_tot, q_mer := s_tot.Query(ptile), s_mer.Query(ptile)
		t.Logf("%0.2f,%v,%v,%v", ptile, q_tot, q_mer, probit(ptile))
	}
}
