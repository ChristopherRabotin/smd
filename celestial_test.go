package smd

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

func TestCelestialObject(t *testing.T) {
	for _, object := range []CelestialObject{Sun, Earth, Mars} {
		object.HelioOrbit(time.Now().UTC())
	}
}

func TestPanics(t *testing.T) {
	assertPanic(t, func() {
		fake := CelestialObject{"Fake", -1, -1, -1, -1, -1, -1, -1, nil}
		fake.HelioOrbit(time.Now())
	})
	assertPanic(t, func() {
		venus := CelestialObject{"Venus", -1, -1, -1, -1, -1, -1, -1, nil}
		venus.HelioOrbit(time.Now())
	})
}

func TestHelio(t *testing.T) {
	dt := time.Date(2017, 03, 20, 14, 45, 0, 0, time.UTC)
	h1 := Earth.HelioOrbit(dt)
	h2 := Earth.HelioOrbit(dt.Add(time.Duration(1) * time.Minute))
	if math.Abs(norm(h1.GetR())-norm(h2.GetR())) > 1e2 {
		t.Fatal("radius changed by more than 100 km in a minute")
	}
	if math.Abs(norm(h1.GetV())-norm(h2.GetV())) > 1e-4 {
		t.Fatal("velocity changed by more than 1 m/s in a minute")
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
	sc := NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, []*Cargo{cargo}, waypoints)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(7*24) * time.Hour)      // Propagate for 7 days.

	sc.LogInfo()
	conf := ExportConfig{Filename: "Rugg", OE: true, Cosmo: true, Timestamp: false}
	astro := NewMission(sc, initOrbit, start, end, false, conf)
	astro.Propagate()

	// Delete the output files.
	os.Remove(fmt.Sprintf("%s/orbital-elements-%s-0.csv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/prop-%s-0.xyzv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/orbital-elements-%s-1.csv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/prop-%s-1.xyzv", os.Getenv("DATAOUT"), conf.Filename))
	os.Remove(fmt.Sprintf("%s/catalog-%s.json", os.Getenv("DATAOUT"), conf.Filename))
}
