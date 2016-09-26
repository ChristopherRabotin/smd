package dynamics

import (
	"integrator"
	"math"
	"time"

	"github.com/soniakeys/meeus/elliptic"
)

/* Handles the astrodynamics. */
//e, i, a, Omega, omega, nu

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	a float64 // Semimajor axis, a, in km
	e float64 // Eccentricity, e
	i float64 // Inclination, i, in radians
	ω float64 // Argument of perihelion, ω, in radians
	Ω float64 // Longitude of ascending node, Ω, in radians
	ν float64 // True anomaly, in radians
	μ float64 // Gravitational constant of the center of orbit.
}

// NewOE returns an Orbit definition.
func NewOE(a, e, i, ω, Ω, ν, μ float64) *Orbit {
	return &Orbit{a, e, i, ω, Ω, ν, μ}
}

// NewOEFromRV returns orbital elements from the R and V vectors.
func NewOEFromRV(R, V [3]float64) *Orbit {
	return nil
}

// GetRV returns the R and V vectors in the ECFI frame.
func (o *Orbit) GetRV() (R, V [3]float64) {
	p := o.a * (1.0 - math.Pow(o.e, 2)) // semi-parameter
	// Compute R and V in the perifocal frame (PQW).
	R[0] = p * math.Cos(o.ν) / (1 + o.e*math.Cos(o.ν))
	R[1] = p * math.Sin(o.ν) / (1 + o.e*math.Cos(o.ν))
	R[2] = 0
	V[0] = -math.Sqrt(o.μ/p) * math.Sin(o.ν)
	V[1] = math.Sqrt(o.μ/p) * (o.e + math.Cos(o.ν))
	V[2] = 0
	// TODO: compute rotation.
	return
}

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
	return nil
	//return &Astrocodile{c, s, oe, start, end, start, make(bool, 1), 1e-6}
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	integrator.NewRK4(0, a.stepSize, a)
}

// Stop implements the stop call of the integrator.
func (a *Astrocodile) Stop(i uint64) bool {
	// TODO: Add the waiting on the channel and block it.
	// Possibly need an attribute and a listening goroutine.
	return true
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
