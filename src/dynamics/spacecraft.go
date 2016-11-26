package dynamics

import (
	"fmt"
	"math"
	"os"
	"time"

	kitlog "github.com/go-kit/kit/log"
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
	FuncQ     []func()
	logger    kitlog.Logger
}

// SCLogInit initializes the logger.
func SCLogInit(name string) kitlog.Logger {
	klog := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	klog = kitlog.NewContext(klog).With("spacecraft", name)
	return klog
}

// LogInfo logs the information of this spacecraft.
func (sc *Spacecraft) LogInfo() {
	var wpInfo string
	for i, wp := range sc.WayPoints {
		if i > 0 {
			wpInfo += " -> "
		}
		wpInfo += wp.String()
	}
	sc.logger.Log("level", "notice", "subsys", "astro", "waypoint", wpInfo)
}

// Mass returns the given vehicle mass based on the provided UTC date time.
func (sc *Spacecraft) Mass(dt time.Time) (m float64) {
	m = sc.DryMass
	if sc.FuelMass > 0 {
		m += sc.FuelMass // Only add the fuel mass if it isn't negative!
	}
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
		Δv, reached := wp.ThrustDirection(*o, dt)
		if reached {
			sc.logger.Log("level", "notice", "subsys", "astro", "waypoint", wp.String(), "status", "completed", "r (km)", norm(o.R), "v (km/s)", norm(o.V))
			// Handle waypoint action
			if action := wp.Action(); action != nil {
				switch action.Type {
				case ADDCARGO:
					sc.FuncQ = append(sc.FuncQ, func() {
						action.Cargo.Arrival = dt // Set the arrival date.
						sc.Cargo = append(sc.Cargo, action.Cargo)
						sc.logger.Log("level", "info", "subsys", "adcs", "cargo", "added", "mass", sc.Mass(dt))
					})
					break
				case DROPCARGO:
					initLen := len(sc.Cargo)
					for i, c := range sc.Cargo {
						if c == action.Cargo {
							if len(sc.Cargo) == 1 {
								sc.FuncQ = append(sc.FuncQ, func() {
									sc.Cargo = []*Cargo{}
								})
								break
							}
							sc.FuncQ = append(sc.FuncQ, func() {
								// Replace the found cargo with the last of the list.
								sc.Cargo[i] = sc.Cargo[len(sc.Cargo)-1]
								// Truncate the list
								sc.Cargo = sc.Cargo[:len(sc.Cargo)-1]
							})
							break
						}
					}
					if initLen == len(sc.Cargo) {
						sc.logger.Log("level", "critical", "subsys", "adcs", "cargo", "not found")
					} else {
						sc.logger.Log("level", "info", "subsys", "adcs", "cargo", "dropped", "mass", sc.Mass(dt))
					}
					break
				case REFEARTH:
					sc.FuncQ = append(sc.FuncQ, func() {
						sc.logger.Log("level", "notice", "subsys", "astro", "nowOrbiting", "Earth", "time", dt.String())
						o.ToXCentric(Earth, dt)
					})
					break
				case REFMARS:
					sc.FuncQ = append(sc.FuncQ, func() {
						sc.logger.Log("level", "notice", "subsys", "astro", "nowOrbiting", "Mars", "time", dt.String())
						o.ToXCentric(Mars, dt)
					})
					break
				case REFSUN:
					sc.FuncQ = append(sc.FuncQ, func() {
						sc.logger.Log("level", "notice", "subsys", "astro", "nowOrbiting", "Sun", "time", dt.String())
						o.ToXCentric(Sun, dt)
					})
					break
				default:
					panic("unknown action")
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
			panic(fmt.Errorf("Δv = %+v! Normalization not implemented yet:", Δv))
		}
		for _, thruster := range sc.Thrusters {
			voltage, power := thruster.Max()
			if err := sc.EPS.Drain(voltage, power, dt); err == nil {
				// Okay to thrust.
				tThrust, isp := thruster.Thrust(voltage, power)
				thrust += tThrust
				fuel += tThrust / (isp * 9.807)
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
	return &Spacecraft{name, float64(mass), 0, nil, []Thruster{}, []*Cargo{}, []Waypoint{}, []func(){}, SCLogInit(name)}
}

// NewSpacecraft returns a spacecraft with initialized function queue and logger.
func NewSpacecraft(name string, dryMass, fuelMass float64, eps EPS, prop []Thruster, payload []*Cargo, wp []Waypoint) *Spacecraft {
	return &Spacecraft{name, dryMass, fuelMass, eps, prop, payload, wp, make([]func(), 5), SCLogInit(name)}
}

// Cargo defines a piece of cargo with arrival date and destination orbit
type Cargo struct {
	Arrival time.Time // Time of arrival onto the tug
	*Spacecraft
}
