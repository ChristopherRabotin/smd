package dynamics

import (
	"fmt"
	"math"
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
	ThrustDirection(Orbit, time.Time) (ThrustControl, bool)
	String() string
}

// NewOutwardSpiral defines a new outward spiral from a celestial object.
func NewOutwardSpiral(body CelestialObject, action *WaypointAction) *ReachDistance {
	if action != nil && action.Type == REFSUN {
		// This is handled by the SetState function of Mission and the propagator
		// will crash if there are multiple attempts to switch to another
		action = nil
	}
	return &ReachDistance{body.SOI, action, false}
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

// ThrustDirection implements the Waypoint interface.
func (wp *Loiter) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	dv := Coast{}
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

// ReachDistance is a type of waypoint which thrusts until a given distance is reached from the central body.
type ReachDistance struct {
	distance float64
	action   *WaypointAction
	cleared  bool
}

// String implements the Waypoint interface.
func (wp *ReachDistance) String() string {
	return fmt.Sprintf("Reach distance of %.1f km.", wp.distance)
}

// Cleared implements the Waypoint interface.
func (wp *ReachDistance) Cleared() bool {
	return wp.cleared
}

// ThrustDirection implements the Waypoint interface.
func (wp *ReachDistance) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	if o.GetRNorm() >= wp.distance {
		wp.cleared = true
		return Coast{}, true
	}
	return Tangential{}, false
}

// Action implements the Waypoint interface.
func (wp *ReachDistance) Action() *WaypointAction {
	return wp.action
}

// NewReachDistance defines a new spiral until a given distance is reached.
func NewReachDistance(distance float64, action *WaypointAction) *ReachDistance {
	return &ReachDistance{distance, action, false}
}

// ReachVelocity is a type of waypoint which thrusts until a given velocity is reached from the central body.
type ReachVelocity struct {
	velocity float64
	action   *WaypointAction
	epsilon  float64 // acceptable error in km/s
	cleared  bool
}

// String implements the Waypoint interface.
func (wp *ReachVelocity) String() string {
	return fmt.Sprintf("Reach velocity of %.1f km/s.", wp.velocity)
}

// Cleared implements the Waypoint interface.
func (wp *ReachVelocity) Cleared() bool {
	return wp.cleared
}

// ThrustDirection implements the Waypoint interface.
func (wp *ReachVelocity) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	velocity := norm(o.GetV())
	if math.Abs(velocity-wp.velocity) < wp.epsilon {
		wp.cleared = true
		return Coast{}, true
	}
	if velocity < wp.velocity {
		// Increase velocity if the SC isn't going fast enough.
		return Tangential{}, false
	}
	// Decrease velocity if the SC is going too fast.
	return AntiTangential{}, false

}

// Action implements the Waypoint interface.
func (wp *ReachVelocity) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// NewReachVelocity defines a new spiral until a given velocity is reached.
func NewReachVelocity(velocity float64, action *WaypointAction) *ReachVelocity {
	return &ReachVelocity{velocity, action, 5, false}
}

// OrbitTarget allows to target an orbit.
type OrbitTarget struct {
	target  Orbit
	ctrl    *OptimalΔOrbit
	action  *WaypointAction
	cleared bool
}

// String implements the Waypoint interface.
func (wp *OrbitTarget) String() string {
	return fmt.Sprintf("targeting orbit")
}

// Cleared implements the Waypoint interface.
func (wp *OrbitTarget) Cleared() bool {
	return wp.cleared
}

// Action implements the Waypoint interface.
func (wp *OrbitTarget) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// ThrustDirection implements the optimal orbit target.
func (wp *OrbitTarget) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	if ok, err := wp.target.Equals(o); ok {
		wp.cleared = true
	} else if wp.ctrl.cleared {
		fmt.Printf("[WARNING] OrbitTarget reached @%s *but* %s: %s\n", dt, err, o.String())
		wp.cleared = true
	}
	return wp.ctrl, wp.cleared
}

// NewOrbitTarget defines a new orbit target.
func NewOrbitTarget(target Orbit, action *WaypointAction, meth ControlLawType, laws ...ControlLaw) *OrbitTarget {
	if target.GetPeriapsis() < target.Origin.Radius || target.GetApoapsis() < target.Origin.Radius {
		fmt.Printf("[WARNING] Target orbit on collision course with %s\n", target.Origin)
	}
	return &OrbitTarget{target, NewOptimalΔOrbit(target, meth, laws), action, false}
}
