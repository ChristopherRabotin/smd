package smd

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/gonum/floats"
	"github.com/soniakeys/meeus/julian"
)

func TestCelestialObject(t *testing.T) {
	for _, object := range []CelestialObject{Sun, Venus, Earth, Mars, Jupiter} {
		object.HelioOrbit(time.Now().UTC())
		var i uint8
		for i = 1; i < 6; i++ {
			if i == 2 && object.J(i) != object.J2 {
				t.Fatalf("J2 not returned for %s", object)
			} else if i == 3 && object.J(i) != object.J3 {
				t.Fatalf("J3 not returned for %s", object)
			} else if i == 4 && object.J(i) != object.J4 {
				t.Fatalf("J4 not returned for %s", object)
			} else if (i < 2 || i > 4) && object.J(i) != 0 {
				t.Fatalf("J(%d) = %f != 0 for %s", i, object.J(i), object)
			} else {
				t.Logf("[OK] J(%d) %s", i, object)
			}
		}
	}
}

func TestPanics(t *testing.T) {
	assertPanic(t, func() {
		fake := CelestialObject{"Fake", -1, -1, -1, -1, -1, -1, -1, -1, -1, nil}
		fake.HelioOrbit(time.Now())
	})
	assertPanic(t, func() {
		venus := CelestialObject{"Vesta", -1, -1, -1, -1, -1, -1, -1, -1, -1, nil}
		venus.HelioOrbit(time.Now())
	})
}

func TestHelio(t *testing.T) {
	t.Skip("WARNING: Skipping test helio because I'm unsure about the values in the test.")
	dt := time.Date(2017, 03, 20, 14, 45, 0, 0, time.UTC)
	h1 := Earth.HelioOrbit(dt)
	h2 := Earth.HelioOrbit(dt.Add(time.Duration(1) * time.Minute))
	if math.Abs(Norm(h1.R())-Norm(h2.R())) > 1e2 {
		t.Fatal("radius changed by more than 100 km in a minute")
	}
	if math.Abs(Norm(h1.V())-Norm(h2.V())) > 1e-4 {
		t.Fatal("velocity changed by more than 1 m/s in a minute")
	}

	// The following position and velocities are from Dr. Davis' Lambert test cases.
	for _, exp := range []struct {
		jde  float64
		R, V []float64
		body CelestialObject
	}{{2455450, []float64{147084764.9, -32521189.65, 467.1900914}, []float64{5.94623924, 28.97464121, -0.0007159151471}, Earth},
		{2455610, []float64{-88002509.16, -62680223.13, 4220331.525}, []float64{20.0705936, -28.68982987, -1.551291815}, Venus},
		{2456300, []float64{170145121.3, -117637192.8, -6642044.272}, []float64{14.70149986, 22.00292904, 0.1001095617}, Mars},
		{2457500, []float64{-803451694.7, 121525767.1, 17465211.78}, []float64{-2.110465959, -12.31199244, 0.09819840772}, Jupiter},
		{2460545, []float64{130423562.1, -76679031.85, 3624.816561}, []float64{14.61294123, 25.56747613, -0.0015034455}, Earth},
		{2460919, []float64{19195371.67, 106029328.4, 348953.802}, []float64{-34.57913611, 6.064190776, 2.078550651}, Venus},
	} {
		R, V := exp.body.HelioOrbit(julian.JDToTime(exp.jde)).RV()
		errDis := 3e3  // 3000 km
		errVel := 1e-1 // 0.1 km/
		for i := 0; i < 3; i++ {
			if !floats.EqualWithinAbs(R[i], exp.R[i], errDis) {
				t.Logf("delta[%d] = %f km", i, math.Abs(R[i]-exp.R[i]))
				t.Fatalf("invalid R[%d] for %s @ %s (%f)\ngot %+v\nexp %+v", i, exp.body, julian.JDToTime(exp.jde), exp.jde, R, exp.R)
			}
			if !floats.EqualWithinAbs(V[i], exp.V[i], errVel) {
				t.Logf("delta[%d] = %f km/s", i, math.Abs(V[i]-exp.V[i]))
				t.Fatalf("invalid V[%d] for %s @ %s (%f)\ngot %+v\nexp %+v", i, exp.body, julian.JDToTime(exp.jde), exp.jde, V, exp.V)
			}
		}
	}
}

func TestCosmoBodyChange(t *testing.T) {
	ω := 10.0 // Made up
	Ω := 5.0  // Made up
	ν := 1.0  // I don't care about that guy.

	initOrbit := NewOrbitFromOE(350+Earth.Radius, 0.01, 46, Ω, ω, ν, Earth)

	/* Building spacecraft */
	eps := NewUnlimitedEPS()
	thrusters := []EPThruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	cargo := &Cargo{time.Now(), NewEmptySC("cargo0", 50)}
	ref2sun := WaypointAction{Type: REFSUN, Cargo: cargo}
	endLoiter := WaypointAction{Type: DROPCARGO, Cargo: nil}
	waypoints := []Waypoint{
		NewOutwardSpiral(Earth, &ref2sun),
		NewLoiter(time.Duration(12)*time.Hour, &endLoiter),
	}
	sc := NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, false, []*Cargo{cargo}, waypoints)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(7*24) * time.Hour)      // Propagate for 7 days.

	sc.LogInfo()
	conf := ExportConfig{Filename: "Rugg", AsCSV: true, Cosmo: true, Timestamp: false}
	astro := NewMission(sc, initOrbit, start, end, Perturbations{}, false, conf)
	astro.Propagate()

	// Delete the output files.
	os.Remove(fmt.Sprintf("%s/orbital-elements-%s-0.csv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/prop-%s-0.xyzv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/orbital-elements-%s-1.csv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/prop-%s-1.xyzv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/catalog-%s.json", os.Getenv("DATAOUT"), conf.Filename))
}

func TestMeeus(t *testing.T) {
	meeusconfig := smdConfig()
	meeusconfig.meeus = true
	config = meeusconfig
	R := Earth.HelioOrbit(julian.JDToTime(2456346.2539)).R()
	exp := []float64{-0.146377664880867e8, -1.485144921336979e8, -0.000000771092830e8}
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(R[i], exp[i], 1e-6) {
			t.Fatalf("delta[%d] = %f km", i, math.Abs(R[i]-exp[i]))
		}
	}
}
