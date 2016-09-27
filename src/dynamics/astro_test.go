package dynamics

import (
	"fmt"
	"math"
	"testing"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

const (
	eps = 1e-8
)

func floatEqual(a, b float64) (bool, error) {
	diff := math.Abs(a - b)
	if diff < eps {
		return true, nil
	}
	return false, fmt.Errorf("difference of %3.6f", diff)
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
	if ok, err := floatEqual(a0, a1); !ok {
		t.Fatalf("semi major axis invalid: %s", err)
	}
	if ok, err := floatEqual(e0, e1); !ok {
		t.Fatalf("eccentricity invalid: %s", err)
	}
	if ok, err := floatEqual(i0, i1); !ok {
		t.Fatalf("inclination invalid: %s", err)
	}
	if ok, err := floatEqual(ω0, ω1); !ok {
		t.Fatalf("argument of perigee invalid: %s", err)
	}
	if ok, err := floatEqual(Ω0, Ω1); !ok {
		t.Fatalf("RAAN invalid: %s", err)
	}
	if ok, err := floatEqual(ν0, ν1); !ok {
		t.Fatalf("true anomaly invalid: %s", err)
	}
}

func TestTwoBodyProp(t *testing.T) {
	// Must define some items still.
	//o := NewOrbitFromOE(Earth.Radius+400, 0.1, deg2rad(36), deg2rad(10), deg2rad(5), 0, Earth.μ)
}
