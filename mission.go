package smd

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ChristopherRabotin/ode"
	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// Propagator defines the different propagation methods available.
type Propagator uint8

func (p Propagator) String() string {
	switch p {
	case GaussianVOP:
		return "GVOP"
	case Cartesian:
		return "Cartesian"
	default:
		panic("unknown propagator")
	}
}

const (
	// StepSize is the default step size when propagating an orbit.
	StepSize = 10 * time.Second
	// GaussianVOP propagator fails for circular, equatorial and hyperbolic orbits
	GaussianVOP Propagator = iota + 1
	// Cartesian propagator works in all cases
	Cartesian
)

var wg sync.WaitGroup

/* Handles the astrodynamical propagations. */

// Mission defines a mission and does the propagation.
type Mission struct {
	Vehicle        *Spacecraft // As pointer because SC may be altered during propagation.
	Orbit          *Orbit      // As pointer because the orbit changes during propagation.
	StartDT        time.Time
	EndDT          time.Time
	CurrentDT      time.Time
	propMethod     Propagator
	perts          Perturbations
	stopChan       chan (bool)
	histChan       chan<- (MissionState)
	done, collided bool
}

// NewMission returns a new Mission instance from the position and velocity vectors.
func NewMission(s *Spacecraft, o *Orbit, start, end time.Time, method Propagator, perts Perturbations, conf ExportConfig) *Mission {
	// If no filepath is provided, then no output will be written.
	var histChan chan (MissionState)
	if !conf.IsUseless() {
		histChan = make(chan (MissionState), 1000) // a 1k entry buffer
		wg.Add(1)
		go func() {
			defer wg.Done()
			StreamStates(conf, histChan)
		}()
	} else {
		histChan = nil
	}
	// Must switch to UTC as all ephemeris data is in UTC.
	if start.Location() != time.UTC {
		start = start.UTC()
	}
	if end.Location() != time.UTC {
		end = end.UTC()
	}

	a := &Mission{s, o, start, end, start, method, perts, make(chan (bool), 1), histChan, false, false}
	// Write the first data point.
	if histChan != nil {
		histChan <- MissionState{a.CurrentDT, *s, *o}
	}

	if end.Before(start) {
		a.Vehicle.logger.Log("level", "warning", "subsys", "astro", "message", "no end date")
	}

	return a
}

// LogStatus returns the status of the propagation and vehicle.
func (a *Mission) LogStatus() {
	a.Vehicle.logger.Log("level", "info", "subsys", "astro", "date", a.CurrentDT, "fuel(kg)", a.Vehicle.FuelMass, "orbit", a.Orbit)
}

// Propagate starts the propagation.
func (a *Mission) Propagate() {
	// Add a ticker status report based on the duration of the simulation.
	a.LogStatus()
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for _ = range ticker.C {
			if a.done {
				break
			}
			a.LogStatus()
		}
	}()
	vInit := norm(a.Orbit.V())
	ode.NewRK4(0, StepSize.Seconds(), a).Solve() // Blocking.
	vFinal := norm(a.Orbit.V())
	a.done = true
	duration := a.CurrentDT.Sub(a.StartDT)
	durStr := duration.String()
	if duration.Hours() > 24 {
		durStr += fmt.Sprintf(" (~%.1fd)", duration.Hours()/24)
	}
	a.Vehicle.logger.Log("level", "notice", "subsys", "astro", "status", "finished", "duration", durStr, "Δv(km/s)", math.Abs(vFinal-vInit))
	a.LogStatus()
	if a.Vehicle.FuelMass < 0 {
		a.Vehicle.logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", a.Vehicle.FuelMass)
	}
	wg.Wait() // Don't return until we're done writing all the files.
}

// StopPropagation is used to stop the propagation before it is completed.
func (a *Mission) StopPropagation() {
	a.stopChan <- true
}

// Stop implements the stop call of the integrator. To stop the propagation, call StopPropagation().
func (a *Mission) Stop(t float64) bool {
	select {
	case <-a.stopChan:
		if a.histChan != nil {
			close(a.histChan)
		}
		return true // Stop because there is a request to stop.
	default:
		a.CurrentDT = a.CurrentDT.Add(StepSize)
		if a.EndDT.Before(a.StartDT) {
			// Check if any waypoint still needs to be reached.
			for _, wp := range a.Vehicle.WayPoints {
				if !wp.Cleared() {
					return false
				}
			}
			if a.histChan != nil {
				close(a.histChan)
			}
			return true
		}
		if a.CurrentDT.Sub(a.EndDT).Nanoseconds() > 0 {
			if a.histChan != nil {
				close(a.histChan)
			}
			return true // Stop, we've reached the end of the simulation.
		}
	}
	return false
}

// GetState returns the state for the integrator for the Gaussian VOP.
func (a *Mission) GetState() (s []float64) {
	s = make([]float64, 7)
	switch a.propMethod {
	case GaussianVOP:
		s[0] = a.Orbit.a
		s[1] = a.Orbit.e
		s[2] = a.Orbit.i
		s[3] = a.Orbit.Ω
		s[4] = a.Orbit.ω
		s[5] = a.Orbit.ν
	case Cartesian:
		R, V := a.Orbit.RV()
		s[0] = R[0]
		s[1] = R[1]
		s[2] = R[2]
		s[3] = V[0]
		s[4] = V[1]
		s[5] = V[2]
	default:
		panic("propagator not implemented")
	}
	s[6] = a.Vehicle.FuelMass
	return
}

// SetState sets the updated state.
func (a *Mission) SetState(t float64, s []float64) {
	if a.histChan != nil {
		a.histChan <- MissionState{a.CurrentDT, *a.Vehicle, *a.Orbit}
	}

	switch a.propMethod {
	case GaussianVOP:
		for i := 2; i <= 5; i++ {
			if s[i] < 0 {
				s[i] += 2 * math.Pi
			}
			s[i] = math.Mod(s[i], 2*math.Pi)
		}

		a.Orbit.a = s[0]
		a.Orbit.e = math.Abs(s[1]) // eccentricity is always a positive number
		a.Orbit.i = s[2]
		a.Orbit.Ω = s[3]
		a.Orbit.ω = s[4]
		a.Orbit.ν = s[5]
	case Cartesian:
		R := []float64{s[0], s[1], s[2]}
		V := []float64{s[3], s[4], s[5]}
		*a.Orbit = *NewOrbitFromRV(R, V, a.Orbit.Origin)
	default:
		panic("propagator not implemented")
	}

	// Orbit sanity checks and warnings.
	if !a.collided && a.Orbit.RNorm() < a.Orbit.Origin.Radius {
		a.collided = true
		a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "collided", a.Orbit.Origin.Name, "dt", a.CurrentDT)
	} else if a.collided && a.Orbit.RNorm() > a.Orbit.Origin.Radius*1.01 {
		// Now further from the 1% dead zone
		a.collided = false
		a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "revived", a.Orbit.Origin.Name, "dt", a.CurrentDT)
	}

	// Propulsion sanity check
	if a.Vehicle.FuelMass > 0 && s[6] <= 0 {
		a.Vehicle.logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", s[6])
	}
	a.Vehicle.FuelMass = s[6]

	// Let's execute any function which is in the queue of this time step.
	for _, f := range a.Vehicle.FuncQ {
		if f == nil {
			continue
		}
		f()
	}
	a.Vehicle.FuncQ = make([]func(), 5) // Clear the queue.

}

// Func is the integration function using Gaussian VOP as per Ruggiero et al. 2011.
func (a *Mission) Func(t float64, f []float64) (fDot []float64) {
	fDot = make([]float64, 7) // init return vector
	// Let's add the thrust to increase the magnitude of the velocity.
	// XXX: Should this Accelerate call be with tmpOrbit?!
	Δv, usedFuel := a.Vehicle.Accelerate(a.CurrentDT, a.Orbit)
	var tmpOrbit *Orbit
	switch a.propMethod {
	case GaussianVOP:
		// *WARNING*: do not fix the angles here because that leads to errors during the RK4 computation.
		// Instead the angles must be fixed and checked only at in SetState function.
		// Note that we don't use Rad2deg because it forces the modulo on the angles, and we want to avoid this for now.
		tmpOrbit = NewOrbitFromOE(f[0], f[1], f[2]/deg2rad, f[3]/deg2rad, f[4]/deg2rad, f[5]/deg2rad, a.Orbit.Origin)
		h := tmpOrbit.HNorm()
		if floats.EqualWithinAbs(tmpOrbit.e, 1, eccentricityε) {
			tmpOrbit.e += 2 * eccentricityε
		}
		p := tmpOrbit.SemiParameter()
		r := tmpOrbit.RNorm()
		sini, cosi := math.Sincos(tmpOrbit.i)
		sinν, cosν := math.Sincos(tmpOrbit.ν)
		sinζ, cosζ := math.Sincos(tmpOrbit.ω + tmpOrbit.ν)
		fR := Δv[0]
		fS := Δv[1]
		fW := Δv[2]
		// da/dt
		fDot[0] = ((2 * math.Pow(tmpOrbit.a, 2)) / h) * (tmpOrbit.e*sinν*fR + (p/r)*fS)
		// de/dt
		fDot[1] = (p*sinν*fR + fS*((p+r)*cosν+r*tmpOrbit.e)) / h
		// di/dt
		fDot[2] = fW * r * cosζ / h
		// dΩ/dt
		fDot[3] = fW * r * sinζ / (h * sini)
		// dω/dt
		fDot[4] = (-p*cosν*fR+(p+r)*sinν*fS)/(h*tmpOrbit.e) - fDot[3]*cosi
		// dν/dt -- as per Vallado, page 636 (with errata of 4th edition.)
		fDot[5] = h/(r*r) + ((p*cosν*fR)-(p+r)*sinν*fS)/(tmpOrbit.e*h)
		// d(fuel)/dt
		fDot[6] = -usedFuel

	case Cartesian:
		R := []float64{f[0], f[1], f[2]}
		V := []float64{f[3], f[4], f[5]}
		tmpOrbit = NewOrbitFromRV(R, V, a.Orbit.Origin)
		bodyAcc := -tmpOrbit.Origin.μ / math.Pow(tmpOrbit.RNorm(), 3)
		//Δv = PQW2ECI(-a.Orbit.Ω, -a.Orbit.i, -a.Orbit.ω, Δv)
		var R1R3, R3R1R3 mat64.Dense
		var ΔvPrime mat64.Vector
		R1R3.Mul(R1(-a.Orbit.i), R3(-a.Orbit.ArgLatitudeU()))
		R3R1R3.Mul(R3(-a.Orbit.Ω), &R1R3)
		ΔvPrime.MulVec(&R3R1R3, mat64.NewVector(3, Δv))
		for i := 0; i < 3; i++ {
			Δv[i] = ΔvPrime.At(i, 0)
		}

		fDot[0] = f[3]
		fDot[1] = f[4]
		fDot[2] = f[5]
		fDot[3] = bodyAcc*f[0] + Δv[0]
		fDot[4] = bodyAcc*f[1] + Δv[1]
		fDot[5] = bodyAcc*f[2] + Δv[2]
	default:
		panic("propagator not implemented")
	}
	// Compute and add the perturbations (which are method dependent).
	// XXX: Should I be using the temp orbit instead?
	pert := a.perts.Perturb(*tmpOrbit, a.CurrentDT, a.propMethod)

	for i := 0; i < 7; i++ {
		fDot[i] += pert[i]
		if math.IsNaN(fDot[i]) {
			r, v := a.Orbit.RV()
			panic(fmt.Errorf("fDot[%d]=NaN @ dt=%s\ncur:%s\tΔv=%+v\nR=%+v\tV=%+v", i, a.CurrentDT, a.Orbit, Δv, r, v))
		}
	}
	return
}

// MissionState stores propagated state.
type MissionState struct {
	DT    time.Time
	SC    Spacecraft
	Orbit Orbit
}
