package dynamics

import (
	"log"
	"time"
)

// Spacecraft defines a new spacecraft.
type Spacecraft struct {
	Name      string      // Name of spacecraft
	DryMass   float64     // DryMass of spacecraft (in kg)
	FuelMass  float64     // FuelMass of spacecraft (in kg) (will panic if runs out of fuel)
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

// Acceleration returns the acceleration to be applied at a given orbital position.
// Keeps track of the thrust applied by all thrusters, the mass changes, and necessary optmizations.
func (sc *Spacecraft) Acceleration(dt *time.Time, o *Orbit) float64 {
	// Here goes the optimizations based on the available power and whether the goal has been reached.
	thrust := 0.0
	for _, wp := range sc.WayPoints {
		if !wp.cleared {
			if wp.Reached(o) {
				wp.cleared = true
				log.Println("waypoint reached")
				continue // Move on to next waypoint.
			}
			for _, thruster := range sc.Thrusters {
				// TODO: Find a way to optimize the thrust?
				tThrust, tMass := thruster.Thrust(thruster.Max())
				thrust += tThrust
				sc.FuelMass -= tMass
			}
			break
		}
	}
	return thrust / sc.Mass(dt)
}

// NewEmptySC returns a spacecraft with no cargo and no thrusters.
func NewEmptySC(name string, mass uint) *Spacecraft {
	return &Spacecraft{name, float64(mass), 0, []Thruster{}, []*Cargo{}, []*Waypoint{}}
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

// Thruster defines a thruster interface.
type Thruster interface {
	// Returns the minimum power and voltage requirements for this thruster.
	Min() (voltage, power uint)
	// Returns the max power and voltage requirements for this thruster.
	Max() (voltage, power uint)
	// Returns the thrust in Newtons and the fuelMass in kg.
	Thrust(voltage, power uint) (thrust, fuelMass float64)
}

/* Available thrusters */

// PPS1350 is the Snecma thruster used on SMART-1.
type PPS1350 struct{}

// Min implements the Thruster interface.
func (t *PPS1350) Min() (voltage, power uint) {
	return 0, 0
}

// Max implements the Thruster interface.
func (t *PPS1350) Max() (voltage, power uint) {
	return 50, 1200
}

// Thrust implements the Thruster interface.
func (t *PPS1350) Thrust(voltage, power uint) (thrust, fuelMass float64) {
	if voltage == 50 && power == 1200 {
		return 68 * 1e-3, 0
	}
	panic("unsupported voltage or power provided")
}

// HPHET12k5 is based on the NASA & Rocketdyne 12.5kW demo
type HPHET12k5 struct{}

// Min implements the Thruster interface.
func (t *HPHET12k5) Min() (voltage, power uint) {
	return 400, 12500
}

// Max implements the Thruster interface.
func (t *HPHET12k5) Max() (voltage, power uint) {
	return 400, 12500
}

// Thrust implements the Thruster interface.
func (t *HPHET12k5) Thrust(voltage, power uint) (thrust, fuelMass float64) {
	if voltage == 400 && power == 12500 {
		return 0.680, 0
	}
	panic("unsupported voltage or power provided")
}
