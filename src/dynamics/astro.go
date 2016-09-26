package dynamics

import (
	"integrator"
	"time"

	"github.com/soniakeys/meeus/elliptic"
)

/* Handles the astrodynamics. */
//e, i, a, Omega, omega, nu

// Astrocodile is an orbit propagator.
// It's a play on words from STK's Atrogrator.
type Astrocodile struct {
	Center    *CelestialObject
	Vehicle   *Spacecraft
	Orbit     *elliptic.Elements
	StartDT   *time.Time
	EndDT     *time.Time
	CurrentDT *time.Time
	StopChan  <-chan (bool)
	stepSize  float64 // This duplicates information a bit but is needed for the duration.
}

// NewAstroFromRV returns a new Astrocodile instance from the position and velocity vectors.
func NewAstroFromRV(c *CelestialObject, s *Spacecraft, R, V []float64, start, end *time.Time) *Astrocodile {
	return &Astrocodile{c, s, oe, start, end, start, make(bool, 1), 1e-6}
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	integrator.NewRK4(0, a.stepSize, a)
}

func (a *Astrocodile) Stop(i uint64) bool {
	// TODO: Add the waiting on the channel and block it.
	// Possibly need an attribute and a listening goroutine.

}

// GetState returns the state for the integrator.
func (a *Astrocodile) GetState() []float64 {
	return nil
}

// SetState sets the updated state.
func (a *Astrocodile) SetState(i uint64, s []float64) {
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, s []float64) []float64 {
	return nil
}
