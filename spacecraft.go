package smd

import (
	"fmt"
	"math"
	"os"
	"time"

	kitlog "github.com/go-kit/kit/log"
)

// Spacecraft defines a new spacecraft.
type Spacecraft struct {
	Name        string                 // Name of spacecraft
	DryMass     float64                // DryMass of spacecraft (in kg)
	FuelMass    float64                // FuelMass of spacecraft (in kg) (will panic if runs out of fuel)
	EPS         EPS                    // EPS definition, needed for the EPThrusters.
	EPThrusters []EPThruster           // All available EP EPThrusters
	ChemProp    bool                   // Set to true to allow Hohmann Transfers.
	Cargo       []*Cargo               // All onboard cargo
	WayPoints   []Waypoint             // All waypoints of the tug
	Maneuvers   map[time.Time]Maneuver // List of maneuvers.
	FuncQ       []func()
	logger      kitlog.Logger
	prevCL      *ControlLaw // Stores the previous control law to follow what is going on.
	Drag        float64
	handleFuel  bool
}

// SCLogInit initializes the logger.
func SCLogInit(name string) kitlog.Logger {
	klog := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	klog = kitlog.With(klog, "spacecraft", name)
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
	// Refuse massless vehicles.
	if m <= 0 {
		m = 1
	}
	return
}

// Accelerate returns the applied velocity (in km/s) at a given orbital position and date time, and the fuel used.
// Keeps track of the thrust applied by all EPThrusters, with necessary optimizations based on next waypoint, *but*
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
		ctrl, reached := wp.ThrustDirection(*o, dt)
		if clType := ctrl.Type(); sc.prevCL == nil || *sc.prevCL != clType {
			sc.logger.Log("level", "info", "subsys", "astro", "date", dt, "thrust", clType, "reason", ctrl.Reason(), "v(km/s)", Norm(o.V()), "orbit", o, "period", o.Period())
			sc.prevCL = &clType
		}
		if reached {
			sc.logger.Log("level", "notice", "subsys", "astro", "waypoint", wp, "status", "completed", "r(km)", Norm(o.R()), "v (km/s)", Norm(o.V()))
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
					sc.FuncQ = append(sc.FuncQ, sc.ToXCentric(Earth, dt, o))
					break
				case REFMARS:
					sc.FuncQ = append(sc.FuncQ, sc.ToXCentric(Mars, dt, o))
					break
				case REFSUN:
					sc.FuncQ = append(sc.FuncQ, sc.ToXCentric(Sun, dt, o))
					break
				default:
					panic("unknown action")
				}
			}
			continue
		}
		Δv := ctrl.Control(*o)
		// Let's normalize the allocation.
		if ΔvNorm := Norm(Δv); ΔvNorm == 0 {
			// Nothing to do, we're probably just loitering.
			return []float64{0, 0, 0}, 0
		} else if math.Abs(ΔvNorm-1) > 1e-12 {
			panic(fmt.Errorf(" Δv = %+v! Normalization not implemented yet ", Δv))
		}
		for _, EPThruster := range sc.EPThrusters {
			voltage, power := EPThruster.Max()
			if err := sc.EPS.Drain(voltage, power, dt); err == nil {
				// Okay to thrust.
				tThrust, isp := EPThruster.Thrust(voltage, power)
				thrust += tThrust
				fuel += tThrust / (isp * 9.807)
			} // Error handling of EPS happens in EPS subsystem.
		}
		thrust /= sc.Mass(dt) // Convert kg*m/(s^-2) to m/(s^-2)
		thrust /= 1e3         // Convert m/s^-2 to km/s^-2
		// For Chem prop, let's make sure the thrust is not nil.
		if thrust == 0 && sc.ChemProp {
			thrust = 1
		}
		// Apply norm of the thrust to each component of the normalized Δv vector
		Δv[0] *= thrust
		Δv[1] *= thrust
		Δv[2] *= thrust
		return Δv, fuel
	}
	return
}

// ToXCentric switches the propagation from the current origin to a new one and logs the change.
func (sc *Spacecraft) ToXCentric(body CelestialObject, dt time.Time, o *Orbit) func() {
	return func() {
		sc.logger.Log("level", "info", "subsys", "astro", "date", dt, "fuel(kg)", sc.FuelMass, "orbit", o)
		o.ToXCentric(body, dt)
		sc.logger.Log("level", "notice", "subsys", "astro", "date", dt, "orbiting", body.Name)
		sc.logger.Log("level", "notice", "subsys", "astro", "R", fmt.Sprintf("%+v km", o.rVec), "V", fmt.Sprintf("%+v km/s", o.vVec))
		sc.logger.Log("level", "info", "subsys", "astro", "date", dt, "fuel(kg)", sc.FuelMass, "orbit", o)
		sc.LogInfo()
	}
}

// NewEmptySC returns a spacecraft with no cargo and no EPThrusters.
func NewEmptySC(name string, mass uint) *Spacecraft {
	return &Spacecraft{name, float64(mass), 0, NewUnlimitedEPS(), []EPThruster{}, false, []*Cargo{}, []Waypoint{}, make(map[time.Time]Maneuver), []func(){}, SCLogInit(name), nil, 0, false}
}

// NewSpacecraft returns a spacecraft with initialized function queue and logger.
func NewSpacecraft(name string, dryMass, fuelMass float64, eps EPS, prop []EPThruster, impulse bool, payload []*Cargo, wp []Waypoint) *Spacecraft {
	return &Spacecraft{name, dryMass, fuelMass, eps, prop, impulse, payload, wp, make(map[time.Time]Maneuver), make([]func(), 5), SCLogInit(name), nil, 0, fuelMass > 0}
}

// Cargo defines a piece of cargo with arrival date and destination orbit
type Cargo struct {
	Arrival time.Time // Time of arrival onto the tug
	*Spacecraft
}
