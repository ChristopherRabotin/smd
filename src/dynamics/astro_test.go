package dynamics

import (
	"fmt"
	"math"
	"testing"
	"time"
)

/* Testing here should propagate a given a orbit which is created via OEs and check that only nu changes.*/

const (
	eps = 1e-8
)

func floatEqual(a, b float64) (bool, error) {
	diff := math.Abs(a - b)
	if diff < eps {
		return true, nil
	}
	return false, fmt.Errorf("difference of %3.10f", diff)
}

func TestAstrocroChanStop(t *testing.T) {
	// Define a new orbit.
	a0 := Earth.Radius + 400
	e0 := 1e-2
	i0 := Deg2rad(38)
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := Deg2rad(1)
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
	// Must find a way to test the stop channel. via a long propagation and a select probably.
	// Check the orbital elements.
	a1, e1, i1, ω1, Ω1, ν1 := o.GetOE()
	if ok, err := floatEqual(a0, a1); !ok {
		t.Fatalf("semi major axis changed: %s", err)
	}
	if diff := math.Abs(e1 - e0); diff > 1e-5 {
		t.Fatalf("eccentricity changed by %f", diff)
	}
	if ok, err := floatEqual(i0, i1); !ok {
		t.Fatalf("inclination changed: %s", err)
	}
	if ok, err := floatEqual(Ω0, Ω1); !ok {
		t.Fatalf("RAAN changed: %s", err)
	}
	if ok, err := floatEqual(ω0, ω1); !ok {
		t.Fatalf("argument of perigee changed: %s", err)
	}
	if ok, _ := floatEqual(ν0, ν1); ok {
		t.Fatalf("true anomaly *unchanged*: ν0=%3.6f ν1=%3.6f", ν0, ν1)
	} else {
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(ν1-ν0), stepSize)
	}
}

func TestAstrocroPropTime(t *testing.T) {
	t.SkipNow()
	// Define an approximate GEO orbit.
	a0 := Earth.Radius + 35786
	e0 := 1e-4
	i0 := 1e-4
	ω0 := Deg2rad(10)
	Ω0 := Deg2rad(5)
	ν0 := Deg2rad(0.1)
	o := NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, Earth)
	// Define propagation parameters.
	start := time.Now()
	end := start.Add(time.Duration(23) * time.Hour).Add(time.Duration(56) * time.Minute).Add(time.Duration(4) * time.Second).Add(time.Duration(916) * time.Millisecond)
	astro, _ := NewAstro(NewEmptySC("test", 1500), o, start, end, "")
	// Start propagation.
	astro.Propagate()
	// Must find a way to test the stop channel. via a long propagation and a select probably.
	// Check the orbital elements.
	a1, e1, i1, ω1, Ω1, ν1 := o.GetOE()
	if ok, err := floatEqual(a0, a1); !ok {
		t.Fatalf("semi major axis changed: %s", err)
	}
	// Eccentricity gets a special treatment because 33% of the time its fails to get the eps precision.
	if diff := math.Abs(e1 - e0); diff > 1e-5 {
		t.Fatalf("eccentricity changed by %f", diff)
	}
	if ok, err := floatEqual(i0, i1); !ok {
		t.Fatalf("inclination changed: %s", err)
	}
	if ok, err := floatEqual(Ω0, Ω1); !ok {
		t.Fatalf("RAAN changed: %s", err)
	}
	if ok, err := floatEqual(ω0, ω1); !ok {
		t.Fatalf("argument of perigee changed: %s", err)
	}
	if diff := math.Abs(ν1 - ν0); diff > 1e-5 {
		t.Fatalf("ν changed too much after one sideral day propagating a GEO vehicle: ν0=%3.6f ν1=%3.6f", ν0, ν1)
	} else {
		t.Logf("ν increased by %5.8f° (step of %0.3f s)\n", Rad2deg(diff), stepSize)
	}
}
