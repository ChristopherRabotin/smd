package dynamics

import (
	"math"
	"testing"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

func TestTwoBodyProp(t *testing.T) {
	// Must define some items still.
	o := NewOrbitFromOE(Earth.Radius+400, 0.1, 36/180.0*2*math.Pi, ω, Ω, 0, Earth.μ)
}
