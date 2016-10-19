package dynamics

import "time"

// WaypointActionEnum defines the possible waypoint actions.
type WaypointActionEnum uint8

const (
	// ADD is a waypoint action associated to a piece of cargo
	ADD WaypointActionEnum = iota + 1
	// DROP is the opposite of ADD
	DROP
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
}

// OutwardSpiral defines an outward spiral waypoint.
type OutwardSpiral struct {
	distance float64
	action   *WaypointAction
	cleared  bool
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
	return &OutwardSpiral{body.SOI, action, false}
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
