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

func TestPetropoulosCaseA(t *testing.T) {
	t.Skip("Case A fails because Petropoulos not yet implemented")
	oInit := NewOrbitFromOE(7000, 0.01, 0.05, 0, 0, 1, Earth)
	oTarget := NewOrbitFromOE(42000, 0.01, 0.05, 0, 0, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{NewGenericEP(1, 3100)}
	dryMass := 1.0
	fuelMass := 299.0
	sc := NewSpacecraft("Petro", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔaCL, OptiΔeCL)})
	start := time.Now()
	// With eta=0.968, the duration is 152.389 days.
	end := start.Add(time.Duration(153*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) || !floats.EqualWithinAbs(astro.Orbit.e, oTarget.e, eccentricityε) {
		t.Fatalf("\ntarget orbit: %s\nfinal orbit:  %s", oTarget, astro.Orbit)
	}
}

func TestPetropoulosCaseB(t *testing.T) {
	t.Skip("Case B *panics* because Petropoulos not yet implemented")
	/*
			--- FAIL: TestPetropoulosCaseB (10.66s)
		panic: fDot[0]=NaN @ dt=2017-03-03 05:34:49.897963525 +0000 UTC
		p=-29614018966.178040   h=NaN   sin=0.048633    dv=[1.7976900650108415e-07 3.2212905975098715e-08 -1.0897265151164653e-16]
		tmp:a=728041967465511.750 e=1.000 i=0.104 ω=0.541 Ω=359.421 ν=2.788
		cur:a=24975088807143.336 e=1.000 i=5.957 ω=31.023 Ω=326.848 ν=159.701 [recovered]
		        panic: fDot[0]=NaN @ dt=2017-03-03 05:34:49.897963525 +0000 UTC
		p=-29614018966.178040   h=NaN   sin=0.048633    dv=[1.7976900650108415e-07 3.2212905975098715e-08 -1.0897265151164653e-16]
		tmp:a=728041967465511.750 e=1.000 i=0.104 ω=0.541 Ω=359.421 ν=2.788
		cur:a=24975088807143.336 e=1.000 i=5.957 ω=31.023 Ω=326.848 ν=159.701
	*/
	oInit := NewOrbitFromOE(24505.9, 0.725, 7.05, 0, 0, 1, Earth)
	oTarget := NewOrbitFromOE(42165, 0.001, 0.05, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{NewGenericEP(0.350, 2000)}
	dryMass := 1.0
	fuelMass := 1999.0
	sc := NewSpacecraft("Petro", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔaCL, OptiΔeCL, OptiΔiCL)})
	start := time.Now()
	// There is no provided time, but the graph goes all the way to 1000 days.
	end := start.Add(time.Duration(1000*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) || !floats.EqualWithinAbs(astro.Orbit.e, oTarget.e, eccentricityε) || !floats.EqualWithinAbs(astro.Orbit.i, oTarget.i, angleε) {
		t.Fatalf("\ntarget orbit: %s\nfinal orbit:  %s", oTarget, astro.Orbit)
	}
}

func TestPetropoulosCaseC(t *testing.T) {
	t.Skip("Case C fails because Petropoulos not yet implemented")
	oInit := NewOrbitFromOE(9222.7, 0.02, 0.573, 0, 0, 1, Earth)
	oTarget := NewOrbitFromOE(3000, 0.7, 0.573, 0, 1, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{NewGenericEP(9.3, 3100)}
	dryMass := 1.0
	fuelMass := 299.0
	sc := NewSpacecraft("Petro", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil, OptiΔaCL, OptiΔeCL)})
	start := time.Now()
	// There is no provided time, but the graph goes all the way to 1000 days.
	end := start.Add(time.Duration(8*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if !floats.EqualWithinAbs(astro.Orbit.a, oTarget.a, distanceε) || !floats.EqualWithinAbs(astro.Orbit.e, oTarget.e, eccentricityε) {
		t.Fatalf("\ntarget orbit: %s\nfinal orbit:  %s", oTarget, astro.Orbit)
	}
}

func TestPetropoulosCaseE(t *testing.T) {
	t.Skip("Case E *panics* because Petropoulos not yet implemented")
	/*
			--- FAIL: TestPetropoulosCaseE (2.01s)
		panic: fDot[0]=NaN @ dt=2017-01-16 20:47:31.589238985 +0000 UTC
		p=-3017860045.633927    h=NaN   sin=0.086988    dv=[-6.264196339417118e-07 8.308510505025826e-07 1.4459733195441482e-14]
		tmp:a=11953681102430.662 e=1.000 i=0.006 ω=356.515 Ω=3.523 ν=4.990
		cur:a=409584267023.699 e=0.999 i=0.355 ω=160.296 Ω=201.839 ν=285.949 [recovered]
		        panic: fDot[0]=NaN @ dt=2017-01-16 20:47:31.589238985 +0000 UTC
		p=-3017860045.633927    h=NaN   sin=0.086988    dv=[-6.264196339417118e-07 8.308510505025826e-07 1.4459733195441482e-14]
		tmp:a=11953681102430.662 e=1.000 i=0.006 ω=356.515 Ω=3.523 ν=4.990
		cur:a=409584267023.699 e=0.999 i=0.355 ω=160.296 Ω=201.839 ν=285.949
	*/
	oInit := NewOrbitFromOE(24505.9, 0.725, 0.06, 0, 0, 1, Earth)
	oTarget := NewOrbitFromOE(26500, 0.7, 116, 270, 180, 1, Earth)
	eps := NewUnlimitedEPS()
	thrusters := []Thruster{NewGenericEP(2, 2000)}
	dryMass := 1.0
	fuelMass := 1999.0
	sc := NewSpacecraft("Petro", dryMass, fuelMass, eps, thrusters, []*Cargo{}, []Waypoint{NewOrbitTarget(*oTarget, nil)})
	start := time.Now()
	// There is no provided time, but the graph goes all the way to 240 days.
	end := start.Add(time.Duration(240*24) * time.Hour)
	astro := NewAstro(sc, oInit, start, end, ExportConfig{})
	astro.Propagate()
	if ok, err := astro.Orbit.Equals(*oTarget); !ok {
		t.Fatalf("error: %s\ntarget orbit: %s\nfinal orbit:  %s", err, oTarget, astro.Orbit)
	}
}
