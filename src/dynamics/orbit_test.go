package dynamics

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestOrbitRV2COE(t *testing.T) {
	R := []float64{6524.834, 6862.875, 6448.296}
	V := []float64{4.901327, 5.533756, -1.976341}
	o := NewOrbitFromRV(R, V, Earth)
	oT := NewOrbitFromOE(36127.343, 0.832853, 87.870, 227.898, 53.38, 92.335, Earth)
	if ok, err := o.StrictlyEquals(*oT); !ok {
		t.Fatalf("orbits differ: %s", err)
	}
	if ok, err := anglesEqual(Deg2rad(281.282), o.GetTildeω()); !ok {
		t.Fatalf("longitude of periapsis invalid: %s (%f)", err, o.GetTildeω())
	}
	if ok, err := anglesEqual(Deg2rad(145.714), o.GetU()); !ok {
		t.Fatalf("argument of latitude invalid: %s (%f)", err, o.GetU())
	}
	assertPanic(t, func() {
		// We're far from a circular equatorial orbit, so this call should panic
		o.Getλtrue()
	})
}

func TestOrbitCOE2RV(t *testing.T) {
	a0 := 36126.64283
	e0 := 0.83285
	i0 := 87.87
	ω0 := 53.38
	Ω0 := 227.89
	ν0 := 92.335
	R := []float64{6524.344, 6861.535, 6449.125}
	V := []float64{4.902276, 5.533124, -1.975709}

	o0 := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	if !vectorsEqual(R, o0.GetR()) {
		// TODO: Fix this first, via tests from page 114.
		t.Fatalf("R vector incorrectly computed:\n%+v\n%+v", R, o0.GetR())
	}
	if !vectorsEqual(V, o0.GetV()) {
		t.Fatal("V vector incorrectly computed")
	}

	o1 := NewOrbitFromRV(R, V, Earth)
	if ok, err := o0.Equals(*o1); !ok {
		t.Logf("o0: %s\no1: %s", o0, o1)
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

func TestOrbitEquality(t *testing.T) {
	oInit := NewOrbitFromOE(226090298.679, 0.088, 26.195, 3.516, 326.494, 278.358, Sun)
	oTest := NewOrbitFromOE(226090250.608, 0.088, 26.195, 3.516, 326.494, 278.358, Sun)
	if ok, err := oInit.Equals(*oTest); !ok {
		t.Fatalf("orbits not equal: %s", err)
	}
	oTest.ω += math.Pi / 6
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different ω are equal")
	}
	oTest.ω -= math.Pi / 6 // Reset
	oTest.Origin = Earth
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different origins are equal")
	}
}

func TestRadii2ae(t *testing.T) {
	a, e := Radii2ae(4, 2)
	if !floats.EqualWithinAbs(a, 3.0, 1e-12) {
		t.Fatalf("a=%f instead of 3.0", a)
	}
	if !floats.EqualWithinAbs(e, 1/3.0, 1e-12) {
		t.Fatalf("e=%f instead of 1/3", e)
	}
	assertPanic(t, func() {
		Radii2ae(1, 2)
	})
}
