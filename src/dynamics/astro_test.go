package dynamics

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

func TestAstrocroChanStop(t *testing.T) {
	// Define a new orbit.
	a0 := Earth.Radius + 400
	e0 := 1e-2
	i0 := 38.0
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 1.0
	oInit := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(1) * time.Hour)
	astro := NewAstro(NewEmptySC("test", 1500), o, start, end, ExportConfig{})
	// Start propagation.
	go astro.Propagate()
	// Check stopping the propagation via the channel.
	<-time.After(time.Millisecond * 1)
	astro.StopChan <- true
	if astro.EndDT.Sub(astro.CurrentDT).Nanoseconds() <= 0 {
		t.Fatal("WARNING: propagation NOT stopped via channel")
	}
	if ok, err := oInit.Equals(*o); !ok {
		t.Fatalf("1ms propagation changes the orbit: %s", err)
	}
	if floats.EqualWithinAbs(Deg2rad(ν0), o.ν, angleε) {
		t.Fatalf("true anomaly *unchanged*: ν0=%3.6f ν1=%3.6f", Deg2rad(ν0), o.ν)
	} else {
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", o.ν-Deg2rad(ν0), stepSize)
	}
}

func TestAstrocroPropTime(t *testing.T) {
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := 10.0
	Ω0 := 5.0
	ν0 := 0.0
	oInit := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// Define propagation parameters.
	start := time.Now()
	geoDur := (time.Duration(23) * time.Hour) + (time.Duration(56) * time.Minute) + (time.Duration(4) * time.Second) + (time.Duration(916) * time.Millisecond)
	end := start.Add(geoDur * 2)
	astro := NewAstro(NewEmptySC("test", 1500), o, start, end, ExportConfig{})
	// Start propagation.
	astro.Propagate()
	// Must find a way to test the stop channel. via a long propagation and a select probably.
	// Check the orbital elements.
	if ok, err := oInit.Equals(*o); !ok {
		t.Fatalf("1ms propagation changes the orbit: %s", err)
	}
	if ok, err := anglesEqual(o.ν, ν0); !ok {
		t.Fatalf("ν changed too much after one sideral day propagating a GEO vehicle: %s", err)
	} else {
		t.Logf("one sideral day GEO ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(math.Abs(o.ν-Deg2rad(ν0))), stepSize)
	}
	// Check that all angular orbital elements are within 2 pi.
	for k, angle := range []float64{o.i, o.Ω, o.ω, o.ν} {
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
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
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
