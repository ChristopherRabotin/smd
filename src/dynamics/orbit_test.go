package dynamics

import (
	"testing"
	"time"
)

func vectorsEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := len(a) - 1; i >= 0; i-- {
		if ok, _ := floatEqual(a[i], b[i]); !ok {
			return false
		}
	}
	return true
}

func TestOrbitDefinition(t *testing.T) {
	a0 := Earth.Radius + 400
	e0 := 0.1
	i0 := Deg2rad(38)
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := 0.1

	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)

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
	if ok, err := floatEqual(Ω0, Ω1); !ok {
		t.Fatalf("RAAN invalid: %s", err)
	}
	if ok, err := floatEqual(ω0, ω1); !ok {
		t.Fatalf("argument of perigee invalid: %s", err)
	}
	if ok, err := floatEqual(ν0, ν1); !ok {
		t.Fatalf("true anomaly invalid: %s", err)
	}
}

func TestOrbitRefChange(t *testing.T) {
	a0 := Earth.Radius + 400
	e0 := 0.1
	i0 := Deg2rad(38)
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := 0.1

	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	dt := time.Now()
	var earthR1, earthV1, earthR2, earthV2, helioR, helioV [3]float64
	copy(earthR1[:], o.R)
	copy(earthV1[:], o.V)
	o.ToXCentric(Sun, dt)
	copy(helioR[:], o.R)
	copy(helioV[:], o.V)
	if vectorsEqual(helioR[:], earthR1[:]) {
		t.Fatal("helioR == earthR1")
	}
	if vectorsEqual(helioV[:], earthV1[:]) {
		t.Fatal("helioV == earthV1")
	}
	// Revert back to Earth centric
	o.ToXCentric(Earth, dt)
	copy(earthR2[:], o.R)
	copy(earthV2[:], o.V)
	if vectorsEqual(helioR[:], earthR2[:]) {
		t.Fatal("helioR == earthR2")
	}
	if vectorsEqual(helioV[:], earthV2[:]) {
		t.Fatal("helioV == earthV2")
	}
	if !vectorsEqual(earthR1[:], earthR2[:]) {
		t.Fatal("earthR1 != earthR2")
	}
	if !vectorsEqual(earthV1[:], earthV2[:]) {
		t.Fatal("earthV1 != earthV2")
	}
}
