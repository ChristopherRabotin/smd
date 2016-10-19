package dynamics

import (
	"math"
	"time"
)

// Spacecraft defines a new spacecraft.
type Spacecraft struct {
	Name      string     // Name of spacecraft
	DryMass   float64    // DryMass of spacecraft (in kg)
	FuelMass  float64    // FuelMass of spacecraft (in kg) (will panic if runs out of fuel)
	EPS       EPS        // EPS definition, needed for the thrusters.
	Thrusters []Thruster // All available thrusters
	Cargo     []*Cargo   // All onboard cargo
	WayPoints []Waypoint // All waypoints of the tug
}

// Mass returns the given vehicle mass based on the provided UTC date time.
func (sc *Spacecraft) Mass(dt time.Time) (m float64) {
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
func (sc *Spacecraft) Accelerate(dt time.Time, o *Orbit) (Δv []float64, fuel float64) {
	// Here goes the optimizations based on the available power and whether the goal has been reached.
	thrust := 0.0
	fuel = 0.0
	Δv = make([]float64, 3)
	for _, wp := range sc.WayPoints {
		if sc.EPS == nil {
			panic("cannot attempt to reach any waypoint without an EPS")
		}
		if wp.Cleared() {
			continue
		}
		// We've found a waypoint which isn't reached.
		Δv, reached := wp.AllocateThrust(o, dt)
		if reached {
			logger.Log("level", "notice", "subsys", "astro", "waypoint", wp.String())
			// Handle waypoint action
			if action := wp.Action(); action != nil {
				switch action.Type {
				case ADDCARGO:
					action.Cargo.Arrival = dt // Set the arrival date.
					sc.Cargo = append(sc.Cargo, action.Cargo)
					logger.Log("level", "info", "subsys", "adcs", "cargo", "added", "mass", sc.Mass(dt))
					break
				case DROPCARGO:
					initLen := len(sc.Cargo)
					for i, c := range sc.Cargo {
						if c == action.Cargo {
							if len(sc.Cargo) == 1 {
								sc.Cargo = []*Cargo{}
								break
							}
							// Replace the found cargo with the last of the list.
							sc.Cargo[i] = sc.Cargo[len(sc.Cargo)-1]
							// Truncate the list
							sc.Cargo = sc.Cargo[:len(sc.Cargo)-1]
							break
						}
					}
					if initLen == len(sc.Cargo) {
						logger.Log("level", "critical", "subsys", "adcs", "cargo", "not found")
					} else {
						logger.Log("level", "info", "subsys", "adcs", "cargo", "dropped", "mass", sc.Mass(dt))
					}
					break
				case REFEARTH:
				case REFMARS:
				case REFSUN:
					logger.Log("level", "critical", "subsys", "astro", "ref", "not implemented")
				}
			}
			continue
		}
		if Δv[0] == 0 && Δv[1] == 0 && Δv[2] == 0 {
			// Nothing to do, we're probably just loitering.
			return []float64{0, 0, 0}, 0
		}
		// Let's normalize the allocation.
		if ΔvNorm := norm(Δv); ΔvNorm == 0 {
			return []float64{0, 0, 0}, 0
		} else if math.Abs(ΔvNorm-1) > 1e-12 {
			panic("ΔvAlloc does not sum up to 1! Normalization not implemented yet.")
		}
		for _, thruster := range sc.Thrusters {
			voltage, power := thruster.Max()
			if err := sc.EPS.Drain(voltage, power, dt); err == nil {
				// Okay to thrust.
				tThrust, tFuelMass := thruster.Thrust(voltage, power)
				thrust += tThrust
				fuel += tFuelMass
			} // Error handling of EPS happens in EPS subsystem.
		}
		thrust /= 1e3 // Convert thrust from m/s^-2 to km/s^-2
		Δv[0] *= thrust / sc.Mass(dt)
		Δv[1] *= thrust / sc.Mass(dt)
		Δv[2] *= thrust / sc.Mass(dt)

		return Δv, fuel
	}
	return
}

// NewEmptySC returns a spacecraft with no cargo and no thrusters.
func NewEmptySC(name string, mass uint) *Spacecraft {
	return &Spacecraft{name, float64(mass), 0, nil, []Thruster{}, []*Cargo{}, []Waypoint{}}
}

// Cargo defines a piece of cargo with arrival date and destination orbit
type Cargo struct {
	Arrival time.Time // Time of arrival onto the tug
	*Spacecraft
}
