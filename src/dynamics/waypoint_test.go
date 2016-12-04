package dynamics

import (
	"testing"
	"time"
)

func TestOutwardSpiral(t *testing.T) {
	vBody := CelestialObject{"Virtual", 100, -1, 0, 0, 100, 0, nil}
	action := &WaypointAction{ADDCARGO, nil}
	wp := NewOutwardSpiral(vBody, action)
	if wp.Cleared() {
		t.Fatal("Waypoint was cleared at creation.")
	}
	o := *NewOrbitFromRV([]float64{90, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached := wp.ThrustDirection(o, time.Now())
	dV := ctrl.Control(o)
	if reached {
		t.Fatal("Waypoint was reached too early.")
	}
	if norm(dV) == 0 {
		t.Fatal("Waypoint did not lead to any velocity change.")
	}
	if wp.Action() != nil {
		t.Fatal("Waypoint returned an action before being reached.")
	}
	o = *NewOrbitFromRV([]float64{100, 0, 0}, []float64{0, 0, 0}, Sun)
	ctrl, reached = wp.ThrustDirection(o, time.Now())
	dV = ctrl.Control(o)
	if !reached {
		t.Fatal("Waypoint was not reached as it should have been.")
	}
	if norm(dV) != 0 {
		t.Fatal("Reached waypoint still returns a velocity change.")
	}
	if wp.Action() == nil {
		t.Fatal("Waypoint did not return any action after being reached.")
	}
	if len(wp.String()) == 0 {
		t.Fatal("Waypoint string is empty.")
	}
}

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
