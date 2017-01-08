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
	/*
		Add the following possibilities, where:
		the delta V vector for me is f_r, f_theta, f_h.
		+Pinkham's spiral:
		f_r = (q*sqrt(mu*p)*v_r)/(2*r**2)
		f_theta = (q*sqrt(mu*p)*v_theta)/(2*r**2)
		p = p_s * exp*(q0)
		r = (p*(1+q**1))/(1+exp(q0)*(1+q**2)*k*cos(theta-omega))

		+ Lawden's spiral
		r = (r_s*(sin alpha)^6)/(1-3*(sin(alpha))^2)
		theta = theta_0 - 4*alpha-3*cotan(alpha) -- cotan == cot
	*/
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
	if norm(o.GetR()) >= wp.distance {
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
func (wp *ReachEnergy) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	if math.Abs(wp.finalξ-o.Energy()) < math.Abs(0.00001*wp.finalξ) {
		wp.cleared = true
		return Coast{}, true
	}
	if math.Abs(wp.finalξ/o.Energy()) < wp.ratio {
		return AntiTangential{}, false
	}
	return Tangential{}, false
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
// SHould this work differently? Like give a time at which to reach a destination, and do a full opti thrust
// until the orbit is actually that of the destination at the given time. The problem here is that the full
// opti thrust does not work great... But the idea of targetting the planet directly and doing an injection
// when close enough is quite interesting.
type PlanetBound struct {
	destination  CelestialObject
	destSOILower float64
	destSOIUpper float64
	cacheTime    time.Time
	cacheDest    Orbit
	action       *WaypointAction
	cleared      bool
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
1. Thrust all the way until the given planet theoretical SOI if the planet were there,
then slow down if the relative velocity would cause the vehicle to flee, or accelerate
otherwise. Constantly check that the vehicle stays within the theoretical SOI, and
update thrust in consideration.
2. Align argument of periapsis with that of the destination planet.
Then, use the InversionCL in order to thrust only when the planet is on its way to us.
The problem with this is that it may take a while to reach the destination since we
aren't always thrusting.
*/
func (wp *PlanetBound) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	if !o.Origin.Equals(Sun) {
		panic("must be in a heliocentric orbit prior to being PlanetBound")
	}
	// If this is the first call, let's compute the theoretical SOI bounds.
	if wp.destSOILower == wp.destSOIUpper {
		wp.cacheTime = dt
		wp.cacheDest = wp.destination.HelioOrbit(dt)
		wp.destSOILower = norm(wp.cacheDest.GetR()) - wp.destination.SOI
		wp.destSOIUpper = norm(wp.cacheDest.GetR()) + wp.destination.SOI
	}
	var cl ThrustControl
	if math.Abs(o.i-wp.cacheDest.i) > (0.2 / (2 * math.Pi)) {
		// Inclination difference of more than 1 degree, let's change this ASAP since
		// the faster we go, the more energy is needed.
		cl = NewOptimalThrust(OptiΔiCL, "inclination change required")
	} else if norm(o.GetR()) < wp.destSOIUpper {
		// Next if the radius isn't going to hit Mars, increase it until it does.
		//cl = Tangential{"not in theoretical SOI"}
		cl = NewOptimalThrust(OptiΔaCL, "radius not in theoretical SOI")
	} else {
		// Inclination and radius are good. The best would be to find whether the vehicle will
		// hit its apoapsis about when the destination will be there, and if not, change the
		// argument of perigee. and if so, need to circularize the orbit slightly before encounter
		// in order to have a slow relative velocity. This will make the capture easier.

		// We cache the destination helio orbit for a full day to make the simulation faster.
		if wp.cacheTime.After(dt.Add(time.Duration(24) * time.Hour)) {
			wp.cacheTime = dt
			wp.cacheDest = wp.destination.HelioOrbit(dt)
		}

		// We are targeting the theoretical SOI. Let's check if we are within the real SOI.
		rDiff := make([]float64, 3)
		R := o.GetR()
		destR := wp.cacheDest.GetR()
		for i := 0; i < 3; i++ {
			rDiff[i] = R[i] - destR[i]
		}
		if norm(rDiff) < wp.destination.SOI {
			// We are in the SOI, let's do an orbital injection.
			// Note that we return here because we're at destination.
			wp.cleared = true
			return Coast{}, true
		} //else {
		// If the relative velocity is positive, let's slow down.
		//cl = AntiTangential{"going faster than planet"}
		cl = Coast{"waiting for planet"}
		//}
	}
	return cl, false
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
	return &PlanetBound{destination, 0.0, 0.0, time.Unix(0, 0), Orbit{}, action, false}
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
	return &OrbitTarget{target, NewOptimalΔOrbit(target, meth, laws), action, false}
}

// PlanetTarget allows to target an orbit.
type PlanetTarget struct {
	target       Orbit
	ctrl         ThrustControl
	action       *WaypointAction
	destSOILower float64
	destSOIUpper float64
	cleared      bool
	switchedCtrl bool
}

// String implements the Waypoint interface.
func (wp *PlanetTarget) String() string {
	return fmt.Sprintf("targeting %s", wp.target.String())
}

// Cleared implements the Waypoint interface.
func (wp *PlanetTarget) Cleared() bool {
	return wp.cleared
}

// Action implements the Waypoint interface.
func (wp *PlanetTarget) Action() *WaypointAction {
	if wp.cleared {
		return wp.action
	}
	return nil
}

// ThrustDirection implements (inefficently) the optimal orbit target.
func (wp *PlanetTarget) ThrustDirection(o Orbit, dt time.Time) (ThrustControl, bool) {
	//	if r := norm(o.GetR()); r > wp.destSOILower && r < wp.destSOIUpper {
	if ok, _ := wp.target.Equals(o); ok {
		wp.cleared = true
	}
	return wp.ctrl, wp.cleared
}

// NewPlanetTarget defines a new orbit target.
func NewPlanetTarget(body CelestialObject, dt time.Time, action *WaypointAction) *PlanetTarget {
	target := body.HelioOrbit(dt)
	destRAtDT := norm(target.GetR())
	lower := destRAtDT + body.SOI*0.01
	upper := destRAtDT + body.SOI*0.10
	return &PlanetTarget{target, NewOptimalΔOrbit(target, Ruggerio, []ControlLaw{}), action, lower, upper, false, false}
}
