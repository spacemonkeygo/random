// Copyright (C) 2015 Space Monkey, Inc.

package random

// pcg from pcg-random.org
type pcg struct {
	state uint64
	inc   uint64
}

// newPCG constructs a pcg with the given state and inc.
func newPCG(state, inc uint64) pcg {
	const mul = 6364136223846793005

	// this code is equiv to initializing a pcg with a 0 state and the updated
	// inc and running
	//
	//    p.Uint32()
	//    p.state += state
	//    p.Uint32()
	//
	// to get the generator started

	inc = inc<<1 | 1
	return pcg{
		state: (inc+state)*mul + inc,
		inc:   inc,
	}
}

// Uint32 returns a random uint32.
func (p *pcg) Uint32() uint32 {
	const mul = 6364136223846793005

	// this branch will be predicted to be false in most cases and so is
	// essentially free. this causes the zero value of a pcg to be the same as
	// newPCG(0, 0).
	if p.inc == 0 {
		*p = newPCG(0, 0)
	}

	// update the state (LCG step)
	oldstate := p.state
	p.state = oldstate*mul + p.inc

	// apply the output permutation to the old state
	xorshifted := uint32(((oldstate >> 18) ^ oldstate) >> 27)
	rot := uint32(oldstate >> 59)
	return xorshifted>>rot | (xorshifted << ((-rot) & 31))
}

// Intn returns an int uniformly in [0, n)
func (p *pcg) Intn(n int) int {
	return fastMod(p.Uint32(), n)
}

// fastMod computes n % m assuming that n is a random number in the full
// uint32 range.
func fastMod(n uint32, m int) int {
	return int((uint64(n) * uint64(m)) >> 32)
}
