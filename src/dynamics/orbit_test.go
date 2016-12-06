package dynamics

import (
	"testing"
	"time"
)

func TestOrbitDefinition(t *testing.T) {
	a0 := 36127.337764
	e0 := 0.832853
	i0 := 87.870
	ω0 := 53.38
	Ω0 := 227.898
	ν0 := 92.335
	R := []float64{6524.429912390563, 6862.463818182738, 6449.138290037659}
	V := []float64{4.901327, 5.533756, -1.976341}

	o0 := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	if !vectorsEqual(R, o0.GetR()) {
		t.Fatal("R vector incorrectly computed")
	}
	if !vectorsEqual(V, o0.GetV()) {
		t.Fatal("V vector incorrectly computed")
	}

	if ok, err := anglesEqual(Deg2rad(281.27), o0.GetTildeω()); !ok {
		t.Logf("true longitude of perigee invalid: %s", err)
	}

	if ok, err := anglesEqual(Deg2rad(145.60549), o0.GetU()); !ok {
		t.Logf("argument of latitude invalid: %s", err)
	}

	if ok, err := anglesEqual(Deg2rad(55.282587), o0.Getλtrue()); !ok {
		t.Logf("true longitude invalid: %s", err)
	}

	o1 := NewOrbitFromRV(R, V, Earth)
	if ok, err := o0.Equals(*o1); !ok {
		t.Fatal(err)
	}
	if ok, err := anglesEqual(Deg2rad(ν0), o1.ν); !ok {
		t.Fatalf("true anomaly invalid: %s", err)
	}
}

func TestOrbitRefChange(t *testing.T) {
	a0 := 36127.343
	e0 := 0.832853
	i0 := 87.870
	ω0 := 53.38
	Ω0 := 227.898
	ν0 := 92.335

	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	R := o.GetR()
	V := o.GetV()
	dt := time.Date(2016, 03, 01, 0, 0, 0, 0, time.UTC)
	var earthR1, earthV1, earthR2, earthV2, helioR, helioV [3]float64
	copy(earthR1[:], R)
	copy(earthV1[:], V)
	o.ToXCentric(Sun, dt)
	R = o.GetR()
	V = o.GetV()
	copy(helioR[:], R)
	copy(helioV[:], V)
	if vectorsEqual(helioR[:], earthR1[:]) {
		t.Fatal("helioR == earthR1")
	}
	if vectorsEqual(helioV[:], earthV1[:]) {
		t.Fatal("helioV == earthV1")
	}
	// Revert back to Earth centric
	o.ToXCentric(Earth, dt)
	R = o.GetR()
	V = o.GetV()
	copy(earthR2[:], R)
	copy(earthV2[:], V)
	if vectorsEqual(helioR[:], earthR2[:]) {
		t.Fatal("helioR == earthR2")
	}
	if vectorsEqual(helioV[:], earthV2[:]) {
		t.Fatal("helioV == earthV2")
	}
	if !vectorsEqual(earthR1[:], earthR2[:]) {
		t.Logf("r1=%+f", earthR1)
		t.Logf("r2=%+f", earthR2)
		t.Fatal("earthR1 != earthR2")
	}
	if !vectorsEqual(earthV1[:], earthV2[:]) {
		t.Fatal("earthV1 != earthV2")
	}
	// Test panic
	assertPanic(t, func() {
		o.ToXCentric(Earth, dt)
	})
}
