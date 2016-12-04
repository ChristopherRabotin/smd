package dynamics

import (
	"math"
	"testing"
	"time"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

func TestAstrocroChanStop(t *testing.T) {
	// Define a new orbit.
	a0 := Earth.Radius + 400
	e0 := 1e-2
	i0 := Deg2rad(38)
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := Deg2rad(1)
	oInit := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(1) * time.Hour)
	astro, _ := NewAstro(NewEmptySC("test", 1500), o, start, end, "")
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
	if ok, _ := floatEqual(ν0, o.ν); ok {
		t.Fatalf("true anomaly *unchanged*: ν0=%3.6f ν1=%3.6f", ν0, o.ν)
	} else {
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(o.ν-ν0), stepSize)
	}
}

func TestAstrocroPropTime(t *testing.T) {
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := Deg2rad(0.1)
	oInit := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// Define propagation parameters.
	start := time.Now()
	end := start.Add(time.Duration(23) * time.Hour).Add(time.Duration(56) * time.Minute).Add(time.Duration(4) * time.Second).Add(time.Duration(916) * time.Millisecond)
	astro, _ := NewAstro(NewEmptySC("test", 1500), o, start, end, "")
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
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(math.Abs(o.ν-math.Mod(o.ν, 2*math.Pi))), stepSize)
	}
	/*if diff := math.Abs(o.ν - ν0); diff > 1e-5 {
		t.Fatalf("ν changed too much after one sideral day propagating a GEO vehicle: ν0=%3.6f ν1=%3.6f", ν0, o.ν)
	} else {
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(diff), stepSize)
	}*/
}

func TestAstrocroFrame(t *testing.T) {
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := Deg2rad(0.1)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	var R1, V1, R2, V2 [3]float64
	copy(R1[:], o.GetR())
	copy(V1[:], o.GetV())
	// Define propagation parameters.
	start := time.Now()
	end := start.Add(time.Duration(2) * time.Hour)
	astro, _ := NewAstro(NewEmptySC("test", 1500), o, start, end, "")
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
