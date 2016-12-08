package dynamics

import (
	"math"
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
	// Test based on edge case
	a0 := 684420.277672
	e0 := 0.893203
	i0 := 0.174533
	ω0 := 0.474642
	Ω0 := 0.032732
	ν0 := 2.830590

	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// These are two edge cases were cosν is slight below -1 or slightly above +1, leading math.Acos to return NaN.
	// Given the difference is on the order of 1e-18, I suspect this is an approximation error (hence the fix in orbit.go).
	// Let's ensure these edge cases are handled.
	for _, dt := range []time.Time{time.Date(2016, 03, 24, 20, 41, 48, 0, time.UTC),
		time.Date(2016, 04, 14, 20, 50, 23, 0, time.UTC),
		time.Date(2016, 05, 12, 18, 0, 15, 0, time.UTC)} {

		R := o.GetR()
		V := o.GetV()

		var earthR1, earthV1, earthR2, earthV2, helioR, helioV [3]float64
		copy(earthR1[:], R)
		copy(earthV1[:], V)
		o.ToXCentric(Sun, dt)
		R = o.GetR()
		V = o.GetV()
		copy(helioR[:], R)
		copy(helioV[:], V)
		for i := 0; i < 3; i++ {
			if math.IsNaN(R[i]) {
				t.Fatalf("R[%d]=NaN", i)
			}
			if math.IsNaN(V[i]) {
				t.Fatalf("V[%d]=NaN", i)
			}
		}
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
}
