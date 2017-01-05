package dynamics

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

func TestAstrocroStopChan(t *testing.T) {
	// Define a new orbit.
	a0 := Earth.Radius - 1 // Collision test
	e0 := 0.8
	i0 := 38.0
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 1.0
	oInit := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(24) * time.Hour)
	sc := NewEmptySC("test", 1500)
	sc.FuelMass = -1
	astro := NewAstro(sc, o, start, end, ExportConfig{})
	// Start propagation.
	go astro.Propagate()
	// Check stopping the propagation via the channel.
	<-time.After(time.Millisecond * 1)
	astro.StopChan <- true
	if astro.CurrentDT.Equal(astro.StartDT) {
		t.Fatal("astro did *not* propagate time")
	}
	if ok, err := oInit.Equals(*o); !ok {
		t.Fatalf("1ms propagation with no waypoints and no end time changes the orbit: %s", err)
	}
	t.Logf("\noInit: %s\noOscu: %s", oInit, o)
}

func TestAstrocroNegTime(t *testing.T) {
	// Define a new orbit.
	a0 := Earth.Radius - 1 // Collision test
	e0 := 0.8
	i0 := 38.0
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 1.0
	oInit := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(-1) * time.Hour)
	sc := NewEmptySC("test", 1500)
	sc.FuelMass = -1
	astro := NewAstro(sc, o, start, end, ExportConfig{})
	astro.Propagate()
	if astro.CurrentDT.Equal(astro.StartDT) {
		t.Fatal("astro did *not* propagate time")
	}
	if ok, err := oInit.StrictlyEquals(*o); !ok {
		t.Fatalf("1ms propagation with no waypoints and no end time changes the orbit: %s", err)
	}
}

func TestAstrocroGEO(t *testing.T) {
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 0.0
	// Propagating for 1.5 orbits to ensure that time and orbital elements are changed accordingly.
	// Note that the 0.08 is needed because of the int64 truncation of the orbit duration.
	oTgt := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0+180.08, Earth)
	oOsc := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, 0, Earth)
	// Define propagation parameters.
	start := time.Now()
	geoDur := (time.Duration(23) * time.Hour) + (time.Duration(56) * time.Minute) + (time.Duration(4) * time.Second) + (time.Duration(916) * time.Millisecond)
	end := start.Add(time.Duration(float64(geoDur) * 1.5))
	astro := NewAstro(NewEmptySC("test", 1500), oOsc, start, end, ExportConfig{})
	// Start propagation.
	astro.Propagate()
	// Must find a way to test the stop channel. via a long propagation and a select probably.
	// Check the orbital elements.
	if ok, err := oOsc.StrictlyEquals(*oTgt); !ok {
		t.Logf("\noOsc: %s\noTgt: %s", oOsc, oTgt)
		t.Fatalf("GEO 1.5 day propagation leads to incorrect orbit: %s", err)
	}
	// Check that all angular orbital elements are within 2 pi.
	for k, angle := range []float64{oOsc.i, oOsc.Ω, oOsc.ω, oOsc.ν} {
		if !floats.EqualWithinAbs(angle, math.Mod(angle, 2*math.Pi), angleε) {
			t.Fatalf("angle in position %d was not 2*pi modulo: %f != %f rad", k, angle, math.Mod(angle, 2*math.Pi))
		}
	}
}

func TestAstrocroFrame(t *testing.T) {
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 0.0
	o := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0, Earth)
	var R1, V1, R2, V2 [3]float64
	copy(R1[:], o.GetR())
	copy(V1[:], o.GetV())
	// Define propagation parameters.
	start := time.Now()
	end := start.Add(time.Duration(2) * time.Hour)
	astro := NewAstro(NewEmptySC("test", 1500), o, start, end, ExportConfig{})
	// Start propagation.
	astro.Propagate()
	// Check that in this orbit there is a change.
	copy(R2[:], o.GetR())
	copy(V2[:], o.GetV())
	if vectorsEqual(R1[:], R2[:]) {
		t.Fatal("R1 == R2")
	}
	if vectorsEqual(V1[:], V2[:]) {
		t.Fatal("V1 == V2")
	}
}

// TestRuggerioOEa runs the test case from their 2012 conference paper
func TestRuggerioOEa(t *testing.T) {
	oInit := NewOrbitFromOE(24396, 0.001, 0.001, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(42164, 0.001, 0.001, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil)})
	start := time.Now()
	end := start.Add(time.Duration(30.5*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Ruggerio semi-major axis failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 14, 2) {
		t.Fatal("too much fuel used")
	}
}

// TestRuggerioOEi runs the test case from their 2012 conference paper
func TestRuggerioOEi(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+350, 0.001, 46, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+350, 0.001, 51.6, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil)})
	start := time.Now()
	end := start.Add(time.Duration(54*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.i, oTarget.i, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Ruggerio inclination failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 25.8, 2) {
		t.Fatal("too much fuel used")
	}
}

// TestRuggerioOEΩ runs the test case from their 2012 conference paper
func TestRuggerioOEΩ(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 5, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil)})
	start := time.Now()
	end := start.Add(time.Duration(49*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.Ω, oTarget.Ω, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Ruggerio RAAN failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 23.5, 2) {
		t.Fatal("too much fuel used")
	}
}

// TestMultiRuggerio runs the test case from their 2012 conference paper
func TestMultiRuggerio(t *testing.T) {
	oInit := NewOrbitFromOE(24396, 0.7283, 7, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(42164, 0.001, 0.001, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil)})
	start := time.Now()
	end := start.Add(time.Duration(104*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if ok, err := astro.Orbit.Equals(*oTarget); !ok {
		t.Logf("final orbit: \n%s", astro.Orbit)
		t.Fatalf("Ruggerio failed: %s", err)
	}
}
