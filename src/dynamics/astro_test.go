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
	// Propagating for 0.5 orbits to ensure that time and orbital elements are changed accordingly.
	oTgt := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, ν0+180.06, Earth)
	oOsc := NewOrbitFromOE(a0, e0, i0, Ω0, ω0, 0, Earth)
	// Define propagation parameters.
	start := time.Now()
	geoDur := (time.Duration(23) * time.Hour) + (time.Duration(56) * time.Minute) + (time.Duration(4) * time.Second)
	end := start.Add(time.Duration(float64(geoDur) * 0.5))
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

// Note: for the "CorrectOE" tests, the Ruggerio paper does not indicate the mass of the vehicle
// nor the amount of fuel. So I have changed the values to those I find from the specified
// spacecraft so as to detect any change while running the tests.

// TestCorrectOEa runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEa(t *testing.T) {
	oInit := NewOrbitFromOE(24396, 0.001, 0.001, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(42164, 0.001, 0.001, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔaCL)})
	start := time.Now()
	end := start.Add(time.Duration(37*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct semi-major axis failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 17, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 17", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEaNeg runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEaNeg(t *testing.T) {
	oInit := NewOrbitFromOE(42164, 0.001, 0.001, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(24396, 0.001, 0.001, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔaCL)})
	start := time.Now()
	end := start.Add(time.Duration(45*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct semi-major axis failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 21, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 21", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEi runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEi(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+350, 0.001, 46, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+350, 0.001, 51.6, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔiCL)})
	start := time.Now()
	end := start.Add(time.Duration(54*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.i, oTarget.i, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct inclination failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 16, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 16", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEiNeg runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEiNeg(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+350, 0.001, 51.6, 1, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+350, 0.001, 46, 1, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔiCL)})
	start := time.Now()
	end := start.Add(time.Duration(54*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.i, oTarget.i, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct inclination failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 16, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 16", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEΩ runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEΩ(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 5, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔΩCL)})
	start := time.Now()
	end := start.Add(time.Duration(49*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.Ω, oTarget.Ω, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct RAAN failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 16, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 16", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEΩNeg runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEΩNeg(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 5, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔΩCL)})
	start := time.Now()
	end := start.Add(time.Duration(49*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.Ω, oTarget.Ω, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct RAAN failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 16, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 16", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEe runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEe(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+9000, 0.01, 98.7, 0, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+9000, 0.15, 98.7, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔeCL)})
	start := time.Now()
	end := start.Add(time.Duration(30*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.e, oTarget.e, eccentricityε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct eccentricity failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 10, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 10", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEe runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEeNeg(t *testing.T) {
	t.Skip("Making an orbit *less* eccentric fails (no panic)")
	oInit := NewOrbitFromOE(Earth.Radius+9000, 0.15, 98.7, 0, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+9000, 0.01, 98.7, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔeCL)})
	start := time.Now()
	end := start.Add(time.Duration(30*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.e, oTarget.e, eccentricityε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct eccentricity failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 16, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 16", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEω runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEω(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 1, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 6, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔωCL)})
	start := time.Now()
	end := start.Add(time.Duration(49*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.ω, oTarget.ω, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct argument of periapsis failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 0.3, 0.2) {
		t.Fatalf("too much fuel used: %f kg instead of 1", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestCorrectOEωNeg runs the test case from the Ruggerio 2012 conference paper.
func TestCorrectOEωNeg(t *testing.T) {
	oInit := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 6, 1, Earth)
	oTarget := NewOrbitFromOE(Earth.Radius+900, 0.001, 98.7, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{new(PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	sc := NewSpacecraft("Rugg", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔωCL)})
	start := time.Now()
	end := start.Add(time.Duration(49*24) * time.Hour) // just after the expected time
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.ω, oTarget.ω, angleε) {
		t.Logf("\noOsc: %s\noTgt: %s", astro.Orbit, oTarget)
		t.Fatal("Correct argument of periapsis failed")
	}
	if !floats.EqualWithinAbs(fuelMass-astro.Vehicle.FuelMass, 23, 2) {
		t.Fatalf("too much fuel used: %f kg instead of 23", fuelMass-astro.Vehicle.FuelMass)
	}
}

// TestMultiCorrectOE runs the test case from the Ruggerio 2012 conference paper.
func TestMultiCorrectOE(t *testing.T) {
	t.Skip("MultiCorrectOE will panic")
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
		t.Fatalf("Correct failed: %s", err)
	}
}
