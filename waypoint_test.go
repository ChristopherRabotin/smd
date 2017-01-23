package smd

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestLoiter(t *testing.T) {
	action := &WaypointAction{ADDCARGO, nil}
	wp := NewLoiter(time.Duration(1)*time.Minute, action)
	if wp.Cleared() {
		t.Fatal("Waypoint was cleared at creation.")
	}
	initTime := time.Unix(0, 0)
	o := *NewOrbitFromRV([]float64{100, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached := wp.ThrustDirection(o, initTime)
	dV := ctrl.Control(o)
	if reached {
		t.Fatal("Loiter waypoint was reached too early.")
	}
	if norm(dV) != 0 {
		t.Fatal("Loiter waypoint required a velocity change.")
	}
	if wp.Action() != nil {
		t.Fatal("Loiter waypoint returned an action before being reached.")
	}
	o = *NewOrbitFromRV([]float64{100, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached = wp.ThrustDirection(o, initTime.Add(time.Duration(1)*time.Second))
	dV = ctrl.Control(o)
	if reached {
		t.Fatal("Loiter waypoint was reached too early.")
	}
	if norm(dV) != 0 {
		t.Fatal("Loiter waypoint required a velocity change.")
	}
	o = *NewOrbitFromRV([]float64{100, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached = wp.ThrustDirection(o, initTime.Add(time.Duration(1)*time.Minute))
	dV = ctrl.Control(o)
	if !reached {
		t.Fatal("Loiter waypoint was not reached as it should have been.")
	}
	if norm(dV) != 0 {
		t.Fatal("Reached loiter waypoint returned a velocity change after being reached.")
	}
	if wp.Action() == nil {
		t.Fatal("Loiter waypoint did not return any action after being reached.")
	}
	if len(wp.String()) == 0 {
		t.Fatal("Loiter waypoint string is empty.")
	}
}

func TestHohmannΔv(t *testing.T) {
	target := *NewOrbitFromOE(Earth.Radius+35781.34857, 0, 0, 0, 0, 90, Earth)
	oscul := *NewOrbitFromOE(Earth.Radius+191.34411, 0, 0, 0, 0, 90, Earth)
	ΔvApoExp := []float64{0.0, -1.478187, 0.0}
	ΔvPeriExp := []float64{0.0, 2.457038, 0.0}
	tofExp := time.Duration(5)*time.Hour + time.Duration(15)*time.Minute + time.Duration(24)*time.Second

	wp := NewHohmannTransfer(target, nil)
	initDT := time.Now().UTC()
	ctrl, cleared := wp.ThrustDirection(oscul, initDT)
	if cleared {
		t.Fatalf("Hohmann waypoint cleared on initial call")
	}
	Δv0 := ctrl.Control(oscul)
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(ΔvPeriExp[i], Δv0[i], velocityε) {
			t.Fatalf("ΔvPeri[%d] failed: %f != %f", i, ΔvPeriExp[i], Δv0[i])
		}
	}
	// Getting the next Δv, which should be nil.
	oscul.ν += math.Pi / 3.0 // Arbitrary subsequent value
	ctrl1, cleared1 := wp.ThrustDirection(oscul, initDT.Add(time.Duration(1)*time.Hour))
	if cleared1 {
		t.Fatalf("Hohmann waypoint cleared on second call")
	}
	Δv1 := ctrl1.Control(oscul)
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(Δv1[i], 0, velocityε) {
			t.Fatalf("Δv should be nil: %+v", Δv1)
		}
	}
	if wp.ctrl.tof != tofExp {
		t.Fatalf("invalid TOF: %d != %d", wp.ctrl.tof, tofExp)
	}
	// Getting the final Δv, which should be nil.
	oscul.ν += math.Pi / 3.0 // Arbitrary subsequent value
	ctrl2, cleared2 := wp.ThrustDirection(oscul, initDT.Add(tofExp))
	if cleared2 {
		t.Fatalf("Hohmann waypoint cleared on third call")
	}
	Δv2 := ctrl2.Control(oscul)
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(Δv2[i], 0, velocityε) {
			t.Fatalf("ΔvApoExp[%d] failed: %f != %f", i, ΔvApoExp[i], Δv2[i])
		}
	}
	oscul.ν += math.Pi / 3.0 // Arbitrary subsequent value
	// TODO: Fix bug here. The status is not updated to the correct value so the WP does not say that it's clear.
	ctrl3, cleared3 := wp.ThrustDirection(oscul, initDT.Add(time.Duration(1)*time.Hour))
	if !cleared3 {
		t.Fatalf("Hohmann waypoint should be cleared on fourth call")
	}
	Δv3 := ctrl3.Control(oscul)
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(Δv3[i], 0, velocityε) {
			t.Fatalf("Δv should be nil: %+v", Δv1)
		}
	}
}
