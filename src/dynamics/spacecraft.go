package dynamics

import (
	"log"
	"time"
)

var lastAccelerationCall time.Time

// Spacecraft defines a new spacecraft.
type Spacecraft struct {
	Name      string      // Name of spacecraft
	DryMass   float64     // DryMass of spacecraft (in kg)
	FuelMass  float64     // FuelMass of spacecraft (in kg) (will panic if runs out of fuel)
	EPS       EPS         // EPS definition, needed for the thrusters.
	Thrusters []Thruster  // All available thrusters
	Cargo     []*Cargo    // All onboard cargo
	WayPoints []*Waypoint // All waypoints of the tug
}

// Mass returns the given vehicle mass based on the provided UTC date time.
func (sc *Spacecraft) Mass(dt *time.Time) (m float64) {
	m = sc.DryMass + sc.FuelMass
	for _, cargo := range sc.Cargo {
		if dt.After(cargo.Arrival) {
			m += cargo.DryMass
		}
	}
	return
}

// Accelerate returns the applied velocity (in km/s) at a given orbital position and date time, and the fuel used.
// Keeps track of the thrust applied by all thrusters, with necessary optmizations based on next waypoint, *but*
// does not update the fuel available (as it needs to be integrated).
func (sc *Spacecraft) Accelerate(dt *time.Time, o *Orbit) ([]float64, float64) {
	// Here goes the optimizations based on the available power and whether the goal has been reached.
	thrust := 0.0
	usedFuel := 0.0
	for _, wp := range sc.WayPoints {
		if sc.EPS == nil {
			panic("cannot attempt to reach any waypoint without an EPS")
		}
		if !wp.cleared {
			if wp.Reached(o) {
				wp.cleared = true
				log.Println("waypoint reached")
				continue // Move on to next waypoint.
			}
			for _, thruster := range sc.Thrusters {
				// TODO: Find a way to optimize the thrust?
				voltage, power := thruster.Max()
				if err := sc.EPS.Drain(voltage, power, *dt); err == nil {
					// Okay to thrust.
					tThrust, tMass := thruster.Thrust(voltage, power)
					thrust += tThrust
					usedFuel += tMass
				}
			}
			break
		}
	}
	if thrust == 0 {
		return []float64{0, 0, 0}, 0
	}
	// Convert thrust from m/s^-2 to km/s^-2
	thrust /= 1e3
	velocityPolar := Cartesian2Spherical(o.V)
	return Spherical2Cartesian([]float64{thrust / sc.Mass(dt), velocityPolar[1], velocityPolar[2]}), usedFuel
}

// NewEmptySC returns a spacecraft with no cargo and no thrusters.
func NewEmptySC(name string, mass uint) *Spacecraft {
	return &Spacecraft{name, float64(mass), 0, nil, []Thruster{}, []*Cargo{}, []*Waypoint{}}
}

// Waypoint is a waypoint along the way.
// TODO: Must support loitering for a certain time?
type Waypoint struct {
	// Condition is a function which returns whether the current orbit
	// can be considered as marking this waypoint as reached.
	Reached func(position *Orbit) bool
	cleared bool
}

// NewWaypoint returns a new way point.
func NewWaypoint(cond func(position *Orbit) bool) *Waypoint {
	return &Waypoint{cond, false}
}

// Cargo defines a piece of cargo with arrival date and destination orbit
type Cargo struct {
	Arrival     time.Time // Time of arrival onto the tug
	Destination *Waypoint // Destination of cargo
	*Spacecraft
}
