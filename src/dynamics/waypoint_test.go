package dynamics

import (
	"testing"
	"time"
)

func TestOutwardSpiral(t *testing.T) {
	vBody := CelestialObject{"Virtual", 100, 0, 100}
	action := &WaypointAction{ADD, nil}
	wp := NewOutwardSpiral(vBody, action)
	if wp.Cleared() {
		t.Fatal("Waypoint was cleared at creation.")
	}
	dV, reached := wp.AllocateThrust(&Orbit{[]float64{90, 0, 0}, []float64{0, 0, 0}, 0}, time.Now())
	if reached {
		t.Fatal("Waypoint was reached too early.")
	}
	if norm(dV) == 0 {
		t.Fatal("Waypoint did not lead to any velocity change.")
	}
	if wp.Action() != nil {
		t.Fatal("Waypoint returned an action before being reached.")
	}
	dV, reached = wp.AllocateThrust(&Orbit{[]float64{100, 0, 0}, []float64{0, 0, 0}, 0}, time.Now())
	if !reached {
		t.Fatal("Waypoint was not reached as it should have been.")
	}
	if norm(dV) != 0 {
		t.Fatal("Reached waypoint still returns a velocity change.")
	}
	if wp.Action() == nil {
		t.Fatal("Waypoint did not return any action after being reached.")
	}
}

func TestLoiter(t *testing.T) {
	action := &WaypointAction{ADD, nil}
	wp := NewLoiter(time.Duration(1)*time.Minute, action)
	if wp.Cleared() {
		t.Fatal("Waypoint was cleared at creation.")
	}
	initTime := time.Unix(0, 0)
	dV, reached := wp.AllocateThrust(&Orbit{[]float64{0, 0, 0}, []float64{0, 0, 0}, 0}, initTime)
	if reached {
		t.Fatal("Loiter waypoint was reached too early.")
	}
	if norm(dV) != 0 {
		t.Fatal("Loiter waypoint required a velocity change.")
	}
	if wp.Action() != nil {
		t.Fatal("Loiter waypoint returned an action before being reached.")
	}
	dV, reached = wp.AllocateThrust(&Orbit{[]float64{100, 0, 0}, []float64{0, 0, 0}, 0}, initTime.Add(time.Duration(1)*time.Second))
	if reached {
		t.Fatal("Loiter waypoint was reached too early.")
	}
	if norm(dV) != 0 {
		t.Fatal("Loiter waypoint required a velocity change.")
	}
	dV, reached = wp.AllocateThrust(&Orbit{[]float64{100, 0, 0}, []float64{0, 0, 0}, 0}, initTime.Add(time.Duration(1)*time.Minute))
	if !reached {
		t.Fatal("Loiter waypoint was not reached as it should have been.")
	}
	if norm(dV) != 0 {
		t.Fatal("Reached loiter waypoint returned a velocity change after being reached.")
	}
	if wp.Action() == nil {
		t.Fatal("Loiter waypoint did not return any action after being reached.")
	}
}
