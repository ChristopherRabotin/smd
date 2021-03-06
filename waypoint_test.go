package smd

import (
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
	if Norm(dV) != 0 {
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
	if Norm(dV) != 0 {
		t.Fatal("Loiter waypoint required a velocity change.")
	}
	o = *NewOrbitFromRV([]float64{100, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached = wp.ThrustDirection(o, initTime.Add(time.Duration(1)*time.Minute))
	dV = ctrl.Control(o)
	if !reached {
		t.Fatal("Loiter waypoint was not reached as it should have been.")
	}
	if Norm(dV) != 0 {
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

	ΔvFinalExp := -1.478187
	ΔvInitExp := 2.457038
	ΔvThrustingPlus := []float64{1, 0, 0}
	ΔvThrustingMinus := []float64{-1, 0, 0}
	ΔvCoasting := []float64{0, 0, 0}
	tofExp := time.Duration(5)*time.Hour + time.Duration(15)*time.Minute + time.Duration(24)*time.Second
	initDT := time.Date(2017, 1, 20, 12, 13, 14, 15, time.UTC)
	apoDT := initDT.Add(tofExp)
	postApoDT := initDT.Add(tofExp + StepSize)

	assertPanic(t, func() {
		tgt := *NewOrbitFromOE(Earth.Radius+35781.34857, 0.5, 0, 0, 0, 90, Earth)
		NewHohmannTransfer(tgt, nil)
	})

	wp := NewHohmannTransfer(target, nil)

	assertPanic(t, func() {
		osc := *NewOrbitFromOE(Earth.Radius+191.34411, eccentricityε, 0, 0, 0, 180, Earth)
		wp.ThrustDirection(osc, initDT)
	})

	assertPanic(t, func() {
		osc := *NewOrbitFromOE(Earth.Radius+191.34411, 0.5, 0, 0, 0, 90, Earth)
		wp.ThrustDirection(osc, initDT)
	})

	assertPanic(t, func() {
		osc := *NewOrbitFromOE(Earth.Radius+191.34411, eccentricityε, 45, 0, 0, 90, Earth)
		wp.ThrustDirection(osc, initDT)
	})

	ctrl, cleared := wp.ThrustDirection(oscul, initDT)
	if cleared {
		t.Fatalf("Hohmann waypoint cleared on initial call")
	}
	Δv := ctrl.Control(oscul)
	// The Precompute was just called, let's check the values.

	if !floats.EqualWithinAbs(wp.ctrl.ΔvInit, ΔvInitExp, velocityε) {
		t.Fatalf("ΔvInit=%f != %f", wp.ctrl.ΔvInit, ΔvInitExp)
	}

	if !floats.EqualWithinAbs(wp.ctrl.ΔvFinal, ΔvFinalExp, velocityε) {
		t.Fatalf("ΔvFinal=%f != %f", wp.ctrl.ΔvFinal, ΔvFinalExp)
	}

	if !vectorsEqual(Δv, ΔvThrustingPlus) {
		t.Fatalf("expected Hohmann thrusting positively, instead got: %+v", Δv)
	}

	// Let's increase the velocity norm simply to simulate that the initial Δv was applied.
	R, V := oscul.RV()
	V[0] += ΔvInitExp
	oscul = *NewOrbitFromRV(R, V, oscul.Origin)
	ctrl, cleared = wp.ThrustDirection(oscul, initDT)
	if cleared {
		t.Fatalf("Hohmann waypoint cleared on second call")
	}
	Δv = ctrl.Control(oscul)
	if !vectorsEqual(Δv, ΔvCoasting) {
		t.Fatalf("expected Hohmann coasting, instead got: %+v", Δv)
	}

	// Let's increase the date to when we are supposed to do the final burn.
	ctrl, cleared = wp.ThrustDirection(oscul, apoDT)
	if cleared {
		t.Fatalf("Hohmann waypoint cleared on third call")
	}
	Δv = ctrl.Control(oscul)
	if !vectorsEqual(Δv, ΔvThrustingMinus) {
		t.Fatalf("expected Hohmann thrusting negatively, instead got: %+v", Δv)
	}

	// Let's increase the velocity norm simply to simulate that the final Δv was applied.
	R, V = oscul.RV()
	V[0] += ΔvFinalExp
	oscul = *NewOrbitFromRV(R, V, oscul.Origin)
	ctrl, cleared = wp.ThrustDirection(oscul, apoDT)
	if cleared {
		t.Fatalf("Hohmann waypoint cleared on fourth call")
	}
	Δv = ctrl.Control(oscul)
	if !vectorsEqual(Δv, ΔvCoasting) {
		t.Fatalf("expected Hohmann coasting, instead got: %+v", Δv)
	}

	// Let's increase the date to when we are supposed to do the final burn.
	ctrl, cleared = wp.ThrustDirection(oscul, postApoDT)
	if !cleared {
		t.Fatalf("Hohmann waypoint cleared on final call")
	}
	Δv = ctrl.Control(oscul)
	if !vectorsEqual(Δv, ΔvCoasting) {
		t.Fatalf("expected Hohmann coasting, instead got: %+v", Δv)
	}
}

func TestToElliptical(t *testing.T) {
	// Example action
	ref2Mars := WaypointAction{Type: REFMARS, Cargo: nil}
	wp := NewToElliptical(&ref2Mars)
	dt := time.Unix(0, 0)
	// Generate an evident hyperbolic orbit
	o := Earth.HelioOrbit(dt)
	o.ToXCentric(Earth, dt.Add(time.Duration(7*24)*time.Hour))
	ctrl, cleared := wp.ThrustDirection(o, dt)
	if cleared {
		t.Fatal("cleared was true for hyperbolic orbit")
	}
	if ctrl.Type() != antiTangential {
		t.Fatal("expected the control to be antiTangential")
	}
	o = *NewOrbitFromOE(Earth.Radius+191.34411, 0.2, 0, 0, 0, 90, Earth)
	_, cleared = wp.ThrustDirection(o, dt)
	if !cleared {
		t.Fatal("cleared was false for elliptical orbit")
	}
}

func TestToHyperbolic(t *testing.T) {
	// Example action
	ref2Sun := WaypointAction{Type: REFSUN, Cargo: nil}
	wp := NewToHyperbolic(&ref2Sun)
	dt := time.Unix(0, 0)
	o := *NewOrbitFromOE(Earth.Radius+191.34411, 0.2, 0, 0, 0, 90, Earth)
	ctrl, cleared := wp.ThrustDirection(o, dt)
	if cleared {
		t.Fatal("cleared was true for elliptical orbit")
	}
	if ctrl.Type() != tangential {
		t.Fatal("expected the control to be antiTangential")
	}
	// Generate an evident hyperbolic orbit
	o = Earth.HelioOrbit(dt)
	o.ToXCentric(Earth, dt.Add(time.Duration(7*24)*time.Hour))
	_, cleared = wp.ThrustDirection(o, dt)
	if !cleared {
		t.Fatal("cleared was false for hyperbolic orbit")
	}
}
