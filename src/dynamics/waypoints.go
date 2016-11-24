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
	AllocateThrust(Orbit, time.Time) ([]float64, bool)
	String() string
}

// NewOutwardSpiral defines a new outward spiral from a celestial object.
func NewOutwardSpiral(body CelestialObject, action *WaypointAction) *ReachDistance {
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

// AllocateThrust implements the Waypoint interface.
func (wp *Loiter) AllocateThrust(o Orbit, dt time.Time) (dv []float64, reached bool) {
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

// AllocateThrust implements the Waypoint interface.
func (wp *ReachDistance) AllocateThrust(o Orbit, dt time.Time) ([]float64, bool) {
	if norm(o.R) >= wp.distance {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	velocityPolar := Cartesian2Spherical(o.V)
	return Spherical2Cartesian([]float64{1, velocityPolar[1], velocityPolar[2]}), false
}

// Action implements the Waypoint interface.
func (wp *ReachDistance) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
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

// AllocateThrust implements the Waypoint interface.
func (wp *ReachVelocity) AllocateThrust(o Orbit, dt time.Time) ([]float64, bool) {
	velocity := norm(o.V)
	if math.Abs(velocity-wp.velocity) < wp.epsilon {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	velocityPolar := Cartesian2Spherical(o.V)
	if velocity < wp.velocity {
		// Increase velocity if the SC isn't going fast enough.
		return Spherical2Cartesian([]float64{1, velocityPolar[1], velocityPolar[2]}), false
	}
	// Decrease velocity if the SC is going too fast.
	return Spherical2Cartesian([]float64{-1, velocityPolar[1], velocityPolar[2]}), false

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

// ReachEnergy is a type of waypoint allows to allocate a good guess of thrust to reach a given energy.
type ReachEnergy struct {
	finalξ     float64 // Stores the final energy the vehicle should have.
	ratio      float64 // Stores the ratio between the current and final energy at which we switch.
	prevThrust float64
	action     *WaypointAction
	cleared    bool
	started    bool
}

// String implements the Waypoint interface.
func (wp *ReachEnergy) String() string {
	return fmt.Sprintf("Reach energy of %.1f (ratio = %1.1f).", wp.finalξ, wp.ratio)
}

// Cleared implements the Waypoint interface.
func (wp *ReachEnergy) Cleared() bool {
	return wp.cleared
}

// AllocateThrust implements the Waypoint interface.
func (wp *ReachEnergy) AllocateThrust(o Orbit, dt time.Time) ([]float64, bool) {
	if math.Abs(wp.finalξ-o.Energy()) < math.Abs(0.00001*wp.finalξ) {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	velocityPolar := Cartesian2Spherical(o.V)
	thrustDirection := 1.0
	if /*o.Energy() > wp.finalξ ||*/ math.Abs(wp.finalξ/o.Energy()) < wp.ratio {
		// Decelerate
		thrustDirection = -1
	}
	if wp.prevThrust > thrustDirection {
		fmt.Println("Started deceleration")
	} else if wp.prevThrust < thrustDirection {
		fmt.Println("Started acceleration")
	}

	wp.prevThrust = thrustDirection
	return Spherical2Cartesian([]float64{thrustDirection, velocityPolar[1], velocityPolar[2]}), false
}

// Action implements the Waypoint interface.
func (wp *ReachEnergy) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// NewReachEnergy defines a new spiral until a given velocity is reached.
func NewReachEnergy(energy, ratio float64, action *WaypointAction) *ReachEnergy {
	return &ReachEnergy{energy, ratio, 0.0, action, false, false}
}
