package dynamics

import (
	"dataio"
	"integrator"
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
	rVec := Cartesian2Spherical(a.Orbit.R)
	vVec := Cartesian2Spherical(a.Orbit.V)
	s[0] = rVec[0]
	s[1] = rVec[1]
	s[2] = rVec[2]
	s[3] = vVec[0]
	s[4] = vVec[1]
	s[5] = vVec[2]
	return
}

// SetState sets the updated state.
func (a *Astrocodile) SetState(i uint64, s []float64) {
	if a.histChan != nil {
		a.histChan <- &dataio.CgInterpolatedState{JD: julian.TimeToJD(*a.CurrentDT), Position: a.Orbit.R, Velocity: a.Orbit.V}
	}
	a.Orbit.R = Spherical2Cartesian([]float64{s[0], s[1], s[2]})
	a.Orbit.V = Spherical2Cartesian([]float64{s[3], s[4], s[5]})
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, x []float64) (xDot []float64) {
	xDot = make([]float64, 6) // init return vector
	// helpers
	r := x[0]
	//θ := x[1] // Unused?!
	φ := x[2]
	vr := x[3]
	vθ := x[4]
	vφ := x[5]

	xDot[0] = vr
	xDot[1] = vθ / (r * math.Cos(φ))
	xDot[2] = vφ / r
	xDot[3] = (math.Pow(vθ, 2)+math.Pow(vφ, 2))/r - a.Orbit.μ/math.Pow(r, 2)
	xDot[4] = vθ * (vφ*math.Tan(φ) - vr) / r
	xDot[5] = -(vr*vφ + math.Pow(vθ, 2)*math.Tan(φ)) / r

	/*if rNorm <= Earth.Radius {
		log.Printf("[COLLISION WARNING] t=%s |R| = %5.5f", a.CurrentDT, rNorm)
	}*/
	return
}
