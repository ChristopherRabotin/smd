package smd

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestHyperbolicOrbitRV2COE(t *testing.T) {
	// XXX: Check how correct this test data is. It came from the outbound hyperbola from Mars
	// but hyperbolic orbits were very poorly supported before.
	R := []float64{-268699.38507486845, 743304.5626288191, 406170.0480721434}
	V := []float64{-0.905741305869758, 0.22523592084626393, 0.16127777856378084}
	o := NewOrbitFromRV(R, V, Mars)
	a, e, _, _, _, _, _, _, _ := o.Elements()
	if e <= 1 {
		t.Fatalf("e is not greater than 1: %f", e)
	}
	if a >= 0 {
		t.Fatalf("a is positive or nil: %f", a)
	}
}

func TestOrbitRV2COE(t *testing.T) {
	R := []float64{6524.834, 6862.875, 6448.296}
	V := []float64{4.901327, 5.533756, -1.976341}
	o := NewOrbitFromRV(R, V, Earth)
	oT := NewOrbitFromOE(36127.343, 0.832853, 87.869126, 227.898260, 53.384931, 92.335157, Earth)
	t.Logf("ot=%s", oT)
	if ok, err := o.StrictlyEquals(*oT); !ok {
		t.Logf("\no0: %s\no1: %s", o, oT)
		t.Fatalf("orbits differ: %s", err)
	}

	a, e, i, Ω, ω, ν, λ, tildeω, u := oT.Elements()
	i = Rad2deg(i)
	Ω = Rad2deg(Ω)
	ω = Rad2deg(ω)
	ν = Rad2deg(ν)
	λ = Rad2deg(λ)
	u = Rad2deg(u)
	tildeω = Rad2deg(tildeω)

	valladoε := 1e-6
	if !floats.EqualWithinAbs(a, 36127.343, 1e-3) {
		t.Fatalf("incorrect semi major axis=%f", a)
	}
	if !floats.EqualWithinAbs(e, 0.832853, valladoε) {
		t.Fatalf("incorrect eccentricity=%f", e)
	}
	if ok, err := anglesEqual(87.869126, i); !ok {
		t.Fatalf("inclination invalid: %s (%f)", err, i)
	}
	if ok, err := anglesEqual(227.898260, Ω); !ok {
		t.Fatalf("RAAN invalid: %s (%f)", err, Ω)
	}
	if ok, err := anglesEqual(53.384931, ω); !ok {
		t.Fatalf("argument of periapsis invalid: %s (%f)", err, ω)
	}
	if ok, err := anglesEqual(92.335157, ν); !ok {
		t.Fatalf("true anomaly invalid: %s (%f)", err, ν)
	}
	if ok, err := anglesEqual(281.283201, tildeω); !ok {
		t.Fatalf("longitude of periapsis invalid: %s (%f)", err, tildeω)
	}
	if ok, err := anglesEqual(145.720088, u); !ok {
		t.Fatalf("argument of latitude invalid: %s (%f)", err, u)
	}
	if ok, err := anglesEqual(13.618348, λ); !ok {
		t.Fatalf("true longitude invalid: %s (%f)", err, λ)
	}
	if !floats.EqualWithinAbs(o.Energyξ(), -5.516604, valladoε) {
		t.Fatalf("incorrect energy ξ=%f", o.Energyξ())
	}
	if !floats.EqualWithinAbs(norm(o.R()), o.RNorm(), valladoε) {
		t.Fatalf("incorrect r norm |R|=%f\tr=%f", norm(o.R()), o.RNorm())
	}
	if !floats.EqualWithinAbs(norm(o.V()), o.VNorm(), valladoε) {
		t.Fatalf("incorrect v norm |V|=%f\tv=%f", norm(o.V()), o.VNorm())
	}
	if !floats.EqualWithinAbs(norm(o.H()), o.HNorm(), valladoε) {
		t.Fatalf("incorrect h norm |h|=%f\th=%f", norm(o.H()), o.HNorm())
	}
}

func TestOrbitCOE2RV(t *testing.T) {
	a0 := 36126.64283
	e0 := 0.83280
	i0 := 87.874925
	ω0 := 53.378089
	Ω0 := 227.891253
	ν0 := 92.335027
	R := []float64{6524.344, 6861.535, 6449.125}
	V := []float64{4.902276, 5.533124, -1.975709}

	o0 := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	if !vectorsEqual(R, o0.R()) {
		t.Fatalf("R vector incorrectly computed:\n%+v\n%+v", R, o0.R())
	}
	if !vectorsEqual(V, o0.V()) {
		t.Fatal("V vector incorrectly computed")
	}

	o1 := NewOrbitFromRV(R, V, Earth)
	if ok, err := o0.StrictlyEquals(*o1); !ok {
		t.Logf("\no0: %s\no1: %s", o0, o1)
		t.Fatal(err)
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

	o := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	// These are two edge cases were cosν is slight below -1 or slightly above +1, leading math.Acos to return NaN.
	// Given the difference is on the order of 1e-18, I suspect this is an approximation error (hence the fix in orbit.go).
	// Let's ensure these edge cases are handled.
	for _, dt := range []time.Time{time.Date(2016, 03, 24, 20, 41, 48, 0, time.UTC),
		time.Date(2016, 04, 14, 20, 50, 23, 0, time.UTC),
		time.Date(2016, 05, 12, 18, 0, 15, 0, time.UTC)} {

		R := o.R()
		V := o.V()

		var earthR1, earthV1, earthR2, earthV2, helioR, helioV [3]float64
		copy(earthR1[:], R)
		copy(earthV1[:], V)
		o.ToXCentric(Sun, dt)
		R = o.R()
		V = o.V()
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
		R = o.R()
		V = o.V()
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
	oTest := NewOrbitFromOE(226090290.608, 0.088, 26.195, 3.516, 326.494, 278.358, Sun)
	if ok, err := oInit.Equals(*oTest); !ok {
		t.Fatalf("orbits not equal: %s", err)
	}
	oTest = NewOrbitFromOE(226090290.608, 0.088, 26.195, 3.516, 336.494, 278.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different ω are equal")
	}
	oTest = NewOrbitFromOE(226090350.608, 0.088, 26.195, 3.516, 326.494, 278.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different a are equal")
	}
	oTest = NewOrbitFromOE(226090290.608, 0.008, 26.195, 3.516, 326.494, 278.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different e are equal")
	}
	oTest = NewOrbitFromOE(226090290.608, 0.088, 25.195, 3.516, 326.494, 278.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different i are equal")
	}
	oTest = NewOrbitFromOE(226090290.608, 0.088, 26.195, 3.916, 326.494, 278.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different Ω are equal")
	}
	oTest = NewOrbitFromOE(226090290.608, 0.088, 26.195, 3.516, 336.494, 277.358, Sun)
	if ok, _ := oInit.Equals(*oTest); ok {
		t.Fatalf("orbits of different ν are equal")
	}
	oTest = NewOrbitFromOE(226090290.608, 0.088, 26.195, 3.516, 326.494, 278.358, Sun)
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

func TestOrbitΦfpa(t *testing.T) {
	for _, e := range []float64{0.5, 0} {
		for _, ν := range []float64{-120, 120} {
			o := NewOrbitFromOE(1e4, e, 1, 1, 1, ν, Earth)
			if e == 0 {
				// Let's force this to zero because NewOrbitFromOE does an approximation.
				o.cche = 0
			}
			Φ := math.Atan2(o.SinΦfpa(), o.CosΦfpa())
			exp := (ν * e) / 2
			if exp < 0 {
				exp += 360
			}
			if (e != 0 && sign(Φ) != sign(ν)) || !floats.EqualWithinAbs(Rad2deg(Φ), exp, angleε) {
				t.Fatalf("Φ = %f (%f) != %f for e=%f with ν=%f", Rad2deg(Φ), Φ, exp, e, ν)
			}
		}
	}
}

func TestOrbitEccentricAnomaly(t *testing.T) {
	o := NewOrbitFromOE(9567205.5, 0.999, 1, 1, 1, 60, Earth)
	sinE, cosE := o.SinCosE()
	E0 := math.Acos(cosE)
	E1 := math.Asin(sinE)
	E2 := math.Atan2(sinE, cosE)
	if !floats.EqualWithinAbs(E2, E0, angleε) || !floats.EqualWithinAbs(E2, E1, angleε) || !floats.EqualWithinAbs(E2, Deg2rad(1.479658), angleε) {
		t.Fatal("specific value of E incorrect")
	}
	for ν := 0.0; ν < 360.0; ν += 0.1 {
		o1 := NewOrbitFromOE(1e5, 0.2, 1, 1, 1, 60, Earth)
		sinE, cosE = o1.SinCosE()
		_, e, _, _, _, ν, _, _, _ := o1.Elements()
		sinν := sinE * math.Sqrt(1-math.Pow(e, 2)) / (1 - e*cosE)
		cosν := (cosE - e) / (1 - e*cosE)
		ν0 := math.Acos(cosν)
		ν1 := math.Asin(sinν)
		ν2 := math.Atan2(sinν, cosν)
		if !floats.EqualWithinAbs(ν2, ν0, angleε) || !floats.EqualWithinAbs(ν2, ν1, angleε) || !floats.EqualWithinAbs(ν2, ν, angleε) {
			t.Fatalf("computing E failed on ν=%f (cosE=%f\tsinE=%f\tν'=%f')", ν, cosE, sinE, ν0)
		}
	}
}

func TestOrbitSpeCircular(t *testing.T) {
	for _, obj := range []CelestialObject{Earth, Sun, Mars} {
		a := 1.5 * obj.Radius
		e := 1e-7
		i := 25.0
		Ω := 87.0
		ω := 52.0
		ν := 20.5
		oI := NewOrbitFromOE(a, e, i, Ω, ω, ν, obj)
		R, V := oI.RV()
		oV := NewOrbitFromRV(R, V, obj)
		if ok, err := oI.StrictlyEquals(*oV); !ok {
			t.Logf("\noI: %s\noV: %s", oI, oV)
			t.Fatalf("for %s: %s", obj, err)
		}
	}
}

func TestOrbitSpeEquatorial(t *testing.T) {
	for _, obj := range []CelestialObject{Earth, Sun, Mars} {
		a := 1.5 * obj.Radius
		e := .25 //1e-7
		i := 1e-7
		Ω := 87.0
		ω := 52.0
		ν := 20.5
		oI := NewOrbitFromOE(a, e, i, Ω, ω, ν, obj)
		R, V := oI.RV()
		oV := NewOrbitFromRV(R, V, obj)
		if ok, err := oI.StrictlyEquals(*oV); !ok {
			t.Logf("\noI: %s\noV: %s", oI, oV)
			t.Fatalf("for %s: %s", obj, err)
		}
	}
}

func TestOrbitSpeCircularEquatorial(t *testing.T) {
	for _, obj := range []CelestialObject{Earth, Sun, Mars} {
		a := 1.5 * obj.Radius
		e := 1e-7
		i := 1e-7
		Ω := 87.0
		ω := 52.0
		ν := 20.5
		oI := NewOrbitFromOE(a, e, i, Ω, ω, ν, obj)
		R, V := oI.RV()
		oV := NewOrbitFromRV(R, V, obj)
		if ok, err := oI.StrictlyEquals(*oV); !ok {
			t.Logf("\noI: %s\noV: %s", oI, oV)
			t.Fatalf("for %s: %s", obj, err)
		}
	}
}

func TestOrbitRefHelioChange(t *testing.T) {
	// TODO: Add mars test
	/*rInit := []float64{10000, 0, 0}
	vInit := []float64{0, 5, 0}
	o := NewOrbitFromRV(rInit, vInit, Earth)*/
	a0 := 684420.277672
	e0 := 0.893203
	i0 := 0.174533
	ω0 := 0.474642
	Ω0 := 0.032732
	ν0 := 2.830590

	o := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Jupiter)
	var r1, v1, r2, v2 [3]float64
	copy(r1[:], o.rVec)
	copy(v1[:], o.vVec)
	dt := time.Date(2016, 03, 24, 20, 41, 48, 0, time.UTC)

	o.ToXCentric(Sun, dt)
	fmt.Println("=====")
	o.ToXCentric(Earth, dt)
	copy(r2[:], o.rVec)
	copy(v2[:], o.vVec)
	for i := 0; i < 3; i++ {
		r2[i] -= r1[i]
		if !floats.EqualWithinAbs(r2[i], 0, 1e-8) {
			t.Fatalf("element %d of r2 incorrect: %f", i, r2[i])
		}
		v2[i] -= v1[i]
		if !floats.EqualWithinAbs(v2[i], 0, 1e-8) {
			t.Fatalf("element %d of v2 incorrect: %f", i, v2[i])
		}
	}
	oInit := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	if ok, err := oInit.StrictlyEquals(*o); !ok {
		t.Logf("%s", err)
	}
}
