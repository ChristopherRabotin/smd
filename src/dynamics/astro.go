package dynamics

import (
	"dataio"
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
	f = make([]float64, 6) // init return vector
	unitX := []float64{1, 0, 0}
	curR := []float64{s[0], s[1], s[2]}
	rNorm := norm(curR)
	vFactor := -a.Orbit.μ / math.Pow(rNorm, 3)
	// deltaV is the instantenous acceleration, hence a velocity.
	deltaV := []float64{s[0] * vFactor, s[1] * vFactor, s[2] * vFactor}
	/*
			shooting_angle = acosd(dot([X(1) X(2) X(3)], [1 0 0])/ (norm([X(1) X(2) X(3)]) * norm([1 0 0]) ));

		T_vec = [X(7)*cosd(shooting_angle);X(7)*sind(shooting_angle);0]; % We are only looking at the velocity magnitude.
		dv_T = (1/(X(8) + dry_mass)) * T_vec;

	*/
	angle := Rad2deg(math.Acos(dot(curR, unitX) / (rNorm * norm(unitX))))
	thrust := a.Vehicle.Thrust(a.CurrentDT, a.Orbit) / a.Vehicle.Mass(a.CurrentDT)
	tVec := []float64{thrust * math.Cos(Deg2rad(angle)), thrust * math.Sin(Deg2rad(angle)), 0}
	if thrust > 0 {
		// Let's convert the new velocity to spherical coordinates and make sure that we thrust
		// in order to increase the semi-major axis.

		// WARNING:  norm(tVec) == thrust !!!! ==> Actually makes sense...
		log.Printf("Δv = %3.5f km/s (T = %3.5f N)\n", norm(tVec), thrust)
	}
	f[0] = s[3]
	f[1] = s[4]
	f[2] = s[5]
	f[3] = deltaV[0]
	f[4] = deltaV[1]
	f[5] = deltaV[2]
	return
}
