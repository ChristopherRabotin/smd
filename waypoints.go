package smd

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
	return &ReachDistance{body.SOI, action, true, false}
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
	distance         float64
	action           *WaypointAction
	further, cleared bool
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
	if wp.further {
		if o.RNorm() >= wp.distance {
			wp.cleared = true
			return Coast{}, true
		}
		return Tangential{}, false
	}
	if o.RNorm() <= wp.distance {
		wp.cleared = true
		return Coast{}, true
	}
	return AntiTangential{}, false
}

// Action implements the Waypoint interface.
func (wp *ReachDistance) Action() *WaypointAction {
	return wp.action
}

// NewReachDistance defines a new spiral until a given distance is reached.
func NewReachDistance(distance float64, further bool, action *WaypointAction) *ReachDistance {
	return &ReachDistance{distance, action, further, false}
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
	if target.Periapsis() < target.Origin.Radius || target.Apoapsis() < target.Origin.Radius {
		fmt.Printf("[WARNING] Target orbit on collision course with %s\n", target.Origin)
	}
	return &OrbitTarget{target, NewOptimalΔOrbit(target, meth, laws), action, false}
}

// HohmannTransfer allows to perform an Hohmann transfer.
type HohmannTransfer struct {
	action    *WaypointAction
	ctrl      HohmannΔv
	arrivalDT time.Time
	cleared   bool
}

// String implements the Waypoint interface.
func (wp *HohmannTransfer) String() string {
	return fmt.Sprintf("Hohmann transfer")
}

// Cleared implements the Waypoint interface.
func (wp *HohmannTransfer) Cleared() bool {
	return wp.cleared
}

// Action implements the Waypoint interface.
func (wp *HohmannTransfer) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// ThrustDirection implements the optimal orbit target.
func (wp *HohmannTransfer) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	switch wp.ctrl.status {
	case hohmannCompute:
		wp.ctrl.Precompute(o)
		// Update the upcoming status of Hohmann
		wp.ctrl.status = hohmmanInitΔv
		// Initialize the Δv with the current knowledge.
		wp.ctrl.ΔvBurnInit = o.VNorm()
		// Compute the arrivial DT
		wp.arrivalDT = dt.Add(wp.ctrl.tof)
		break
	case hohmmanInitΔv:
		// Nothing to do.
	case hohmmanCoast:
		if dt.After(wp.arrivalDT.Add(-StepSize)) {
			// Next step will be the arrivial DT.
			wp.ctrl.status = hohmmanFinalΔv
			// Initialize the Δv with the current knowledge.
			wp.ctrl.ΔvBurnInit = o.VNorm()
		}
	case hohmmanFinalΔv:
		// Nothing to do.
	case hohmmanCompleted:
		// This state is changed in the control. Hence, the cleared status is only
		// available until the *subsequent* call to ThrustDirection.
		wp.cleared = true
	}
	return &wp.ctrl, wp.cleared
}

// NewHohmannTransfer defines a new Hohmann transfer
func NewHohmannTransfer(target Orbit, action *WaypointAction) *HohmannTransfer {
	if target.Periapsis() < target.Origin.Radius || target.Apoapsis() < target.Origin.Radius {
		fmt.Printf("[WARNING] Target orbit on collision course with %s\n", target.Origin)
	}
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	return &HohmannTransfer{action, NewHohmannΔv(target), epoch, false}
}

// ToElliptical slows down the vehicle until its orbit is elliptical.
type ToElliptical struct {
	action  *WaypointAction
	cleared bool
}

// String implements the Waypoint interface.
func (wp *ToElliptical) String() string {
	return fmt.Sprintf("to elliptical")
}

// Cleared implements the Waypoint interface.
func (wp *ToElliptical) Cleared() bool {
	return wp.cleared
}

// Action implements the Waypoint interface.
func (wp *ToElliptical) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// ThrustDirection implements the optimal orbit target.
func (wp *ToElliptical) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	if o.e < 1 {
		wp.cleared = true
	}
	return AntiTangential{}, wp.cleared
}

// NewToElliptical defines a ToElliptical waypoint.
func NewToElliptical(action *WaypointAction) *ToElliptical {
	return &ToElliptical{action, false}
}
