// Copyright (C) 2018. See AUTHORS.

package random

// coin is a simple struct to let us get random bools and make minimum calls
// to the random number generator.
type coin struct {
	pcg  pcg
	val  uint32
	bits int
}

func (c *coin) toss() (val bool) {
	if c.bits == 0 {
		c.val = c.pcg.Uint32()
		c.bits = 32
	}
	c.bits--
	val = c.val&1 > 0
	c.val >>= 1
	return val
}
