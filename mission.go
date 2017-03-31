package smd

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ChristopherRabotin/ode"
	"github.com/gonum/matrix/mat64"
)

const (
	// StepSize is the default step size of propagation.
	StepSize = 10 * time.Second
)

var wg sync.WaitGroup

/* Handles the astrodynamical propagations. */

// Mission defines a mission and does the propagation.
type Mission struct {
	Vehicle                    *Spacecraft  // As pointer because SC may be altered during propagation.
	Orbit                      *Orbit       // As pointer because the orbit changes during propagation.
	Φ                          *mat64.Dense // STM
	StartDT, StopDT, CurrentDT time.Time
	perts                      Perturbations
	step                       time.Duration // time step
	stopChan                   chan (bool)
	histChan                   chan<- (State)
	computeSTM, done, collided bool
}

// NewMission is the same as NewPreciseMission with the default step size.
func NewMission(s *Spacecraft, o *Orbit, start, end time.Time, perts Perturbations, computeSTM bool, conf ExportConfig) *Mission {
	return NewPreciseMission(s, o, start, end, perts, StepSize, computeSTM, conf)
}

// NewPreciseMission returns a new Mission instance with custom provided time step.
func NewPreciseMission(s *Spacecraft, o *Orbit, start, end time.Time, perts Perturbations, step time.Duration, computeSTM bool, conf ExportConfig) *Mission {
	// If no filepath is provided, then no output will be written.
	var histChan chan (State)
	if !conf.IsUseless() {
		histChan = make(chan (State), 1000) // a 1k entry buffer
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

	a := &Mission{s, o, DenseIdentity(6), start, end, start, perts, step, make(chan (bool), 1), histChan, computeSTM, false, false}
	// Write the first data point.
	if histChan != nil {
		histChan <- State{a.CurrentDT, *s, *o}
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

// PropagateUntil propagates until the given time is reached.
func (a *Mission) PropagateUntil(dt time.Time) {
	a.StopDT = dt
	a.Propagate()
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
	initFuel := a.Vehicle.FuelMass
	ode.NewRK4(0, a.step.Seconds(), a).Solve() // Blocking.
	vFinal := norm(a.Orbit.V())
	a.done = true
	duration := a.CurrentDT.Sub(a.StartDT)
	durStr := duration.String()
	if duration.Hours() > 24 {
		durStr += fmt.Sprintf(" (~%.3fd)", duration.Hours()/24)
	}
	a.Vehicle.logger.Log("level", "notice", "subsys", "astro", "status", "finished", "duration", durStr, "Δv(km/s)", math.Abs(vFinal-vInit), "fuel(kg)", initFuel-a.Vehicle.FuelMass)
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
		a.CurrentDT = a.CurrentDT.Add(a.step)
		if a.StopDT.Before(a.StartDT) {
			// A hard limit is set on a ten year propagation.
			kill := false
			if a.CurrentDT.After(a.StartDT.Add(24 * 3652.5 * time.Hour)) {
				a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "status", "killed")
				kill = true
			}
			if !kill {
				// Check if any waypoint still needs to be reached.
				for _, wp := range a.Vehicle.WayPoints {
					if !wp.Cleared() {
						return false
					}
				}
			}
			if a.histChan != nil {
				close(a.histChan)
			}
			return true
		}
		if a.CurrentDT.Sub(a.StopDT).Nanoseconds() > 0 {
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
	stateSize := 7
	if a.computeSTM {
		stateSize += 36
	}
	s = make([]float64, stateSize)
	R, V := a.Orbit.RV()
	// R, V in the state
	for i := 0; i < 3; i++ {
		s[i] = R[i]
		s[i+3] = V[i]
	}
	s[6] = a.Vehicle.FuelMass
	if a.computeSTM {
		// Add the components of Φ
		sIdx := 6
		for i := 0; i < 6; i++ {
			for j := 0; j < 6; j++ {
				s[sIdx] = a.Φ.At(i, j)
				sIdx++
			}
		}
	}
	return
}

// SetState sets the updated state.
func (a *Mission) SetState(t float64, s []float64) {
	if a.histChan != nil {
		a.histChan <- State{a.CurrentDT, *a.Vehicle, *a.Orbit}
	}

	R := []float64{s[0], s[1], s[2]}
	V := []float64{s[3], s[4], s[5]}
	*a.Orbit = *NewOrbitFromRV(R, V, a.Orbit.Origin) // Deref is important (cd. TestMissionSpiral)

	// Orbit sanity checks and warnings.
	if !a.collided && a.Orbit.RNorm() < a.Orbit.Origin.Radius {
		a.collided = true
		a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "collided", a.Orbit.Origin.Name, "dt", a.CurrentDT, "r", a.Orbit.RNorm(), "radius", a.Orbit.Origin.Radius)
	} else if a.collided && a.Orbit.RNorm() > a.Orbit.Origin.Radius*1.1 {
		// Now further from the 10% dead zone
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

	R := []float64{f[0], f[1], f[2]}
	V := []float64{f[3], f[4], f[5]}
	tmpOrbit = NewOrbitFromRV(R, V, a.Orbit.Origin)
	bodyAcc := -tmpOrbit.Origin.μ / math.Pow(tmpOrbit.RNorm(), 3)
	_, _, i, Ω, _, _, _, _, u := tmpOrbit.Elements()
	Δv = Rot313Vec(-u, -i, -Ω, Δv)
	// d\vec{R}/dt
	fDot[0] = f[3]
	fDot[1] = f[4]
	fDot[2] = f[5]
	// d\vec{V}/dt
	fDot[3] = bodyAcc*f[0] + Δv[0]
	fDot[4] = bodyAcc*f[1] + Δv[1]
	fDot[5] = bodyAcc*f[2] + Δv[2]
	// d(fuel)/dt
	fDot[6] = -usedFuel

	// Compute and add the perturbations (which are method dependent).
	// XXX: Should I be using the temp orbit instead?
	pert := a.perts.Perturb(*tmpOrbit, a.CurrentDT)

	for i := 0; i < 7; i++ {
		fDot[i] += pert[i]
		if math.IsNaN(fDot[i]) {
			r, v := a.Orbit.RV()
			panic(fmt.Errorf("fDot[%d]=NaN @ dt=%s\ncur:%s\tΔv=%+v\nR=%+v\tV=%+v", i, a.CurrentDT, a.Orbit, Δv, r, v))
		}
	}
	return
}

// State stores propagated state.
type State struct {
	DT    time.Time
	SC    Spacecraft
	Orbit Orbit
}
