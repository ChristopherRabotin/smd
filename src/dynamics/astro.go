package dynamics

import (
	"dataio"
	"integrator"
	"math"
	"time"

	"github.com/gonum/matrix/mat64"
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

	a := &Astrocodile{s, o, start, end, start, make(chan (bool), 1), histChan}
	// Write the first data point.
	if histChan != nil {
		histChan <- &dataio.CgInterpolatedState{JD: julian.TimeToJD(*start), Position: a.Orbit.R, Velocity: a.Orbit.V}
	}
	return a
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	integrator.NewRK4(0, stepSize, a).Solve()
	// Add a ticker status report based on the duration of the simulation.
}

// Stop implements the stop call of the integrator.
func (a *Astrocodile) Stop(i uint64) bool {
	select {
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
	s[0] = a.Orbit.R[0]
	s[1] = a.Orbit.R[1]
	s[2] = a.Orbit.R[2]
	s[3] = a.Orbit.V[0]
	s[4] = a.Orbit.V[1]
	s[5] = a.Orbit.V[2]
	return
}

// SetState sets the updated state.
func (a *Astrocodile) SetState(i uint64, s []float64) {
	if a.histChan != nil {
		a.histChan <- &dataio.CgInterpolatedState{JD: julian.TimeToJD(*a.CurrentDT), Position: a.Orbit.R, Velocity: a.Orbit.V}
	}
	a.Orbit.R[0] = s[0]
	a.Orbit.R[1] = s[1]
	a.Orbit.R[2] = s[2]
	a.Orbit.V[0] = s[3]
	a.Orbit.V[1] = s[4]
	a.Orbit.V[2] = s[5]
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, s []float64) (f []float64) {
	vFactor := -a.Orbit.μ / math.Pow(mat64.Norm(mat64.NewVector(3, []float64{s[0], s[1], s[2]}), 2), 3)
	f = make([]float64, 6)
	f[0] = s[3]
	f[1] = s[4]
	f[2] = s[5]
	f[3] = s[0] * vFactor
	f[4] = s[1] * vFactor
	f[5] = s[2] * vFactor
	return
}

// Helper functions go here.

// Radii2ae returns the semi major axis and the eccentricty from the radii.
func Radii2ae(rA, rP float64) (a, e float64) {
	a = (rP + rA) / 2
	e = (rA - rP) / (rA + rP)
	return
}
