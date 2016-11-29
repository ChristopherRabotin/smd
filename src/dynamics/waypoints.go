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
	ThrustDirection(Orbit, time.Time) ([]float64, bool)
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

// ThrustDirection implements the Waypoint interface.
func (wp *Loiter) ThrustDirection(o Orbit, dt time.Time) (dv []float64, reached bool) {
	dv = Coast{}.Control(o)
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
func (wp *ReachDistance) ThrustDirection(o Orbit, dt time.Time) ([]float64, bool) {
	if norm(o.R) >= wp.distance {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	return Inversion{Deg2rad(45)}.Control(o), false
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

// ThrustDirection implements the Waypoint interface.
func (wp *ReachVelocity) ThrustDirection(o Orbit, dt time.Time) ([]float64, bool) {
	velocity := norm(o.V)
	if math.Abs(velocity-wp.velocity) < wp.epsilon {
		wp.cleared = true
		return []float64{0, 0, 0}, true
	}
	if velocity < wp.velocity {
		// Increase velocity if the SC isn't going fast enough.
		return Tangential{}.Control(o), false
	}
	// Decrease velocity if the SC is going too fast.
	return AntiTangential{}.Control(o), false

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
	finalξ  float64 // Stores the final energy the vehicle should have.
	ratio   float64 // Stores the ratio between the current and final energy at which we switch.
	action  *WaypointAction
	cleared bool
	started bool
}

// String implements the Waypoint interface.
func (wp *ReachEnergy) String() string {
	return fmt.Sprintf("Reach energy of %.1f (ratio = %1.1f).", wp.finalξ, wp.ratio)
}

// Cleared implements the Waypoint interface.
func (wp *ReachEnergy) Cleared() bool {
	return wp.cleared
}

// ThrustDirection implements the Waypoint interface.
func (wp *ReachEnergy) ThrustDirection(o Orbit, dt time.Time) ([]float64, bool) {
	if math.Abs(wp.finalξ-o.Energy()) < math.Abs(0.00001*wp.finalξ) {
		wp.cleared = true
		return Coast{}.Control(o), true
	}
	if math.Abs(wp.finalξ/o.Energy()) < wp.ratio {
		return AntiTangential{}.Control(o), false
	}
	return Tangential{}.Control(o), false
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
	return &ReachEnergy{energy, ratio, action, false, false}
}

// PlanetBound is a type of waypoint which thrusts until a given distance is reached from the central body.
type PlanetBound struct {
	destination CelestialObject
	action      *WaypointAction
	cleared     bool
	prevCL      *ControlLaw
}

// String implements the Waypoint interface.
func (wp *PlanetBound) String() string {
	return fmt.Sprintf("Toward Planet %s.", wp.destination.Name)
}

// Cleared implements the Waypoint interface.
func (wp *PlanetBound) Cleared() bool {
	return wp.cleared
}

// ThrustDirection implements the Waypoint interface.
/*
Ideas:
1. Thrust all the way until the given planet theoritical SOI if the planet were there,
then slow down if the relative velocity would cause the vehicle to flee, or accelerate
otherwise. Constantly check that the vehicle stays within the theoritical SOI, and
update thrust in consideration.
2. Align argument of periapsis with that of the destination planet.
Then, use the InversionCL in order to thrust only when the planet is on its way to us.
The problem with this is that it may take a while to reach the destination since we
aren't always thrusting.
*/
func (wp *PlanetBound) ThrustDirection(o Orbit, dt time.Time) ([]float64, bool) {
	if !o.Origin.Equals(Sun) {
		panic("must be in a heliocentric orbit prior to being PlanetBound")
	}
	var cl ThrustControl
	destOrbit := wp.destination.HelioOrbit(dt)
	destSOILower := norm(destOrbit.R) - wp.destination.SOI
	destSOIUpper := norm(destOrbit.R) + wp.destination.SOI
	if r := norm(o.R); r < destSOILower {
		cl = Tangential{}
	} else if r > destSOIUpper {
		cl = AntiTangential{}
	} else {
		// We are in the theoritical SOI. Let's check if we are within the real SOI.
		rDiff := make([]float64, 3)
		for i := 0; i < 3; i++ {
			rDiff[i] = o.R[i] - destOrbit.R[i]
		}
		if norm(rDiff) < wp.destination.SOI {
			// We are in the SOI, let's do an orbital injection.
			// Note that we return here because we're at destination.
			wp.cleared = true
			return Coast{}.Control(o), true
		} else if relVel := norm(o.V) - norm(destOrbit.V); relVel < 0 {
			// If the relative velocity is negative, the planet will catch up with the spacecraft,
			// so let's just wait.
			cl = Coast{}
		} else {
			// If the relative velocity is positive, let's slow down.
			cl = AntiTangential{}
		}
	}
	if clType := cl.Type(); wp.prevCL == nil || *wp.prevCL != clType {
		fmt.Printf("applying %s\n", cl.Type().String())
		wp.prevCL = &clType
	}
	return cl.Control(o), false
}

// Action implements the Waypoint interface.
func (wp *PlanetBound) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// NewPlanetBound defines a new trajectory until and including the orbital insertion.
func NewPlanetBound(destination CelestialObject, action *WaypointAction) *PlanetBound {
	if action == nil || (action.Type != REFEARTH && action.Type != REFMARS) {
		panic("PlanetBound requires a REF* action. ")
	}
	return &PlanetBound{destination, action, false, nil}
}
