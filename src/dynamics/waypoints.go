package dynamics

import (
	"fmt"
	"time"
)

// WaypointActionEnum defines the possible waypoint actions.
type WaypointActionEnum uint8

const (
	// ADDCARGO is a waypoint action associated to a piece of cargo
	ADDCARGO WaypointActionEnum = iota + 1
	// DROPCARGO is the opposite of ADD
	DROPCARGO
	// REFEARTH switches the orbit reference to the Earth
	REFEARTH
	// REFMARS switches the orbit reference to Mars
	REFMARS
	//REFSUN switches the orbit reference to the Sun
	REFSUN
)

// WaypointAction defines what happens when a given waypoint is reached.
type WaypointAction struct {
	Type  WaypointActionEnum
	Cargo *Cargo
}

// Waypoint defines the Waypoint interface.
type Waypoint interface {
	Cleared() bool // returns whether waypoint has been reached
	Action() *WaypointAction
	AllocateThrust(*Orbit, time.Time) ([]float64, bool)
	String() string
}

// OutwardSpiral defines an outward spiral waypoint.
type OutwardSpiral struct {
	distance float64
	action   *WaypointAction
	cleared  bool
	body     string
}

// String implements the Waypoint interface.
func (wp *OutwardSpiral) String() string {
	return fmt.Sprintf("Outward spiral from %s.", wp.body)
}

// Cleared implements the Waypoint interface.
func (wp *OutwardSpiral) Cleared() bool {
	return wp.cleared
}

// AllocateThrust implements the Waypoint interface.
func (wp *OutwardSpiral) AllocateThrust(o *Orbit, dt time.Time) ([]float64, bool) {
	if norm(o.R) >= wp.distance {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	velocityPolar := Cartesian2Spherical(o.V)
	return Spherical2Cartesian([]float64{1, velocityPolar[1], velocityPolar[2]}), false
}

// Action implements the Waypoint interface.
func (wp *OutwardSpiral) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// NewOutwardSpiral defines a new outward spiral from a celestial object.
func NewOutwardSpiral(body CelestialObject, action *WaypointAction) *OutwardSpiral {
	return &OutwardSpiral{body.SOI, action, false, body.Name}
}

// Loiter is a type of waypoint which allows the vehicle to stay at a given position for a given duration.
type Loiter struct {
	duration         time.Duration
	startDT          time.Time
	endDT            time.Time
	startedLoitering bool
	action           *WaypointAction
	cleared          bool
}

// String implements the Waypoint interface.
func (wp *Loiter) String() string {
	return fmt.Sprintf("Coasting for %s.", wp.duration)
}

// Cleared implements the Waypoint interface.
func (wp *Loiter) Cleared() bool {
	return wp.cleared
}

// AllocateThrust implements the Waypoint interface.
func (wp *Loiter) AllocateThrust(o *Orbit, dt time.Time) (dv []float64, reached bool) {
	dv = []float64{0, 0, 0}
	if !wp.startedLoitering {
		// First time this is called, starting timer.
		wp.startedLoitering = true
		wp.startDT = dt
		wp.endDT = dt.Add(wp.duration)
		return dv, false
	}
	if dt.Before(wp.endDT) {
		return dv, false
	}
	wp.cleared = true
	return dv, true
}

// Action implements the Waypoint interface.
func (wp *Loiter) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// NewLoiter defines a new loitering waypoint, i.e. "wait until a given time".
func NewLoiter(duration time.Duration, action *WaypointAction) *Loiter {
	return &Loiter{duration, time.Unix(0, 0), time.Unix(0, 0), false, action, false}
}
