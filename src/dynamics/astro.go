package dynamics

import (
	"dataio"
	"fmt"
	"integrator"
	"log"
	"math"
	"time"

	"github.com/soniakeys/meeus/julian"
)

const (
	stepSize = 1.0
)

/* Handles the astrodynamical propagations. */

// Astrocodile is an orbit propagator.
// It's a play on words from STK's Atrogrator.
type Astrocodile struct {
	Vehicle   *Spacecraft
	Orbit     *Orbit
	StartDT   *time.Time
	EndDT     *time.Time
	CurrentDT *time.Time
	StopChan  chan (bool)
	histChan  chan<- (*dataio.CgInterpolatedState)
	initV     float64
}

// NewAstro returns a new Astrocodile instance from the position and velocity vectors.
func NewAstro(s *Spacecraft, o *Orbit, start, end *time.Time, filepath string) *Astrocodile {
	// If no filepath is provided, then no output will be written.
	var histChan chan (*dataio.CgInterpolatedState)
	if filepath != "" {
		histChan = make(chan (*dataio.CgInterpolatedState), 1000) // a 1k entry buffer
		go dataio.StreamInterpolatedStates(filepath, histChan, false)
	} else {
		histChan = nil
	}

	a := &Astrocodile{s, o, start, end, start, make(chan (bool), 1), histChan, norm(o.V)}
	// Write the first data point.
	if histChan != nil {
		histChan <- &dataio.CgInterpolatedState{JD: julian.TimeToJD(*start), Position: a.Orbit.R, Velocity: a.Orbit.V}
	}
	return a
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	log.Printf("Starting propagation. Simulation start time: %s\n", a.StartDT)
	// Add a ticker status report based on the duration of the simulation.
	var tickDuration time.Duration
	if a.EndDT.Sub(*a.StartDT) > time.Duration(24*30)*time.Hour {
		tickDuration = time.Minute
	} else {
		tickDuration = time.Duration(10) * time.Second
	}
	ticker := time.NewTicker(tickDuration)
	go func() {
		for _ = range ticker.C {
			log.Printf("Simulation time: %s", a.CurrentDT)
		}
	}()
	integrator.NewRK4(0, stepSize, a).Solve() // Blocking.
	log.Printf("Propagation ended. Simulation time: %s\n", a.CurrentDT)
}

// Stop implements the stop call of the integrator.
func (a *Astrocodile) Stop(i uint64) bool {
	select {
	// TODO: Change this to a call to Spacecraft given the orbit information.
	case <-a.StopChan:
		if a.histChan != nil {
			close(a.histChan)
		}
		return true // Stop because there is a request to stop.
	default:
		*a.CurrentDT = a.CurrentDT.Add(time.Duration(stepSize) * time.Second)
		if a.CurrentDT.Sub(*a.EndDT).Nanoseconds() > 0 {
			if a.histChan != nil {
				close(a.histChan)
			}
			return true // Stop, we've reached the end of the simulation.
		}
	}
	return false
}

// GetState returns the state for the integrator.
func (a *Astrocodile) GetState() (s []float64) {
	s = make([]float64, 6)
	for i := 0; i < 3; i++ {
		s[i] = a.Orbit.R[i]
		s[i+3] = a.Orbit.V[i]
	}
	return
}

// SetState sets the updated state.
func (a *Astrocodile) SetState(i uint64, s []float64) {
	if a.histChan != nil {
		a.histChan <- &dataio.CgInterpolatedState{JD: julian.TimeToJD(*a.CurrentDT), Position: a.Orbit.R, Velocity: a.Orbit.V}
	}
	//fmt.Printf("[%s] deltaV = %f km/s\n", a.CurrentDT, math.Abs(norm(a.Orbit.V)-norm([]float64{s[3], s[4], s[5]})))
	a.Orbit.R = []float64{s[0], s[1], s[2]}
	a.Orbit.V = []float64{s[3], s[4], s[5]}
	if norm(a.Orbit.R) < 0 {
		panic("negative distance to body")
	}
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, f []float64) (fDot []float64) {
	fDot = make([]float64, 6) // init return vector
	radius := norm([]float64{f[0], f[1], f[2]})
	if radius < Earth.Radius {
		fmt.Printf("[COLLISION] r=%f km\n", radius)
	}
	// Let's add the thrust to increase the magnitude of the velocity.
	Δv := a.Vehicle.Acceleration(a.CurrentDT, a.Orbit)
	twoBodyVelocity := -a.Orbit.μ / math.Pow(radius, 3)
	for i := 0; i < 3; i++ {
		// The first three components are the velocity.
		fDot[i] = f[i+3]
		// The following three are the instantenous acceleration.
		fDot[i+3] = f[i]*twoBodyVelocity + Δv[i]
	}
	return
}
