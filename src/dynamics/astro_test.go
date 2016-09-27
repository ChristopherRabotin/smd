package dynamics

import (
	"math"
	"testing"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

const (
	eps = 1e-8
)

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < eps
}

func TestOrbitDefinition(t *testing.T) {
	a0 := Earth.Radius + 400
	e0 := 0.1
	i0 := deg2rad(38)
	ω0 := deg2rad(10)
	Ω0 := deg2rad(5)
	ν0 := 0.0

	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth.μ)

	a1, e1, i1, ω1, Ω1, ν1 := o.GetOE()
	if !floatEqual(a0, a1) {
		t.Fatal("semi major axis invalid")
	}
	if !floatEqual(e0, e1) {
		t.Fatal("eccentricity invalid")
	}
	if !floatEqual(i0, i1) {
		t.Fatal("inclination invalid")
	}
	if !floatEqual(ω0, ω1) {
		t.Fatal("ω invalid")
	}
	if !floatEqual(Ω0, Ω1) {
		t.Fatal("Ω invalid")
	}
	if !floatEqual(ν0, ν1) {
		t.Fatal("ν invalid")
	}
}

func TestTwoBodyProp(t *testing.T) {
	// Must define some items still.
	//o := NewOrbitFromOE(Earth.Radius+400, 0.1, deg2rad(36), deg2rad(10), deg2rad(5), 0, Earth.μ)
}
