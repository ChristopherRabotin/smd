package dynamics

import (
	"github.com/soniakeys/meeus/elliptic"
)

/* Handles the astrodynamics. */
//e, i, a, Omega, omega, nu

// Propagate will propagate an orbit from initial orbital elements start until a TRUE is received on the stop channel.
func Propagate(oe *elliptic.Elements, stop <-chan bool) {

}
