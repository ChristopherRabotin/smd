package dynamics

import (
	"integrator"
	"math"
	"time"

	"github.com/gonum/matrix/mat64"
)

/* Handles the astrodynamics. */
//e, i, a, Omega, omega, nu

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	R []float64 // Radius vector
	V []float64 // Velocity vector
	μ float64   // Gravitational constant of the center of orbit.
}

// NewOrbitFromOE creates an orbit from the orbital elements.
func NewOrbitFromOE(a, e, i, ω, Ω, ν, μ float64) *Orbit {
	p := a * (1.0 - math.Pow(e, 2)) // semi-parameter
	R, V := make([]float64, 3), make([]float64, 3)
	// Compute R and V in the perifocal frame (PQW).
	R[0] = p * math.Cos(ν) / (1 + e*math.Cos(ν))
	R[1] = p * math.Sin(ν) / (1 + e*math.Cos(ν))
	R[2] = 0
	V[0] = -math.Sqrt(μ/p) * math.Sin(ν)
	V[1] = math.Sqrt(μ/p) * (e + math.Cos(ν))
	V[2] = 0
	// Compute ECI rotation.
	R = PQW2ECI(i, ω, Ω, R)
	V = PQW2ECI(i, ω, Ω, V)
	return &Orbit{R, V, μ}
}

// NewOrbit returns orbital elements from the R and V vectors. Needed for prop
func NewOrbit(R, V []float64, μ float64) *Orbit {
	return &Orbit{R, V, μ}
}

// GetOE returns the orbital elements of this orbit.
func (o *Orbit) GetOE() (a, e, i, ω, Ω, ν float64) {
	h := []float64{o.R[1]*o.V[2] - o.R[2]*o.V[1],
		o.R[2]*o.V[0] - o.R[0]*o.V[2],
		o.R[0]*o.V[1] - o.R[1]*o.V[0]} // Cross product R x V.

	N := []float64{-o.V[1], o.V[0], 0}

	eVec := make([]float64, 3)
	for j := 0; j < 3; j++ {
		eVec[j] = math.Pow(norm(o.V), 2) - o.R[j]*o.μ/norm(o.R) - dot(o.R, o.V)/o.μ*o.V[j]
	}
	e = norm(eVec) // Eccentricity
	// We suppose the orbit is NOT parabolic.
	a = -o.μ / (2 * (0.5*dot(o.V, o.V) - o.μ/norm(o.V)))
	i = math.Acos(h[2] / norm(h))
	Ω = math.Acos(N[0] / norm(N))

	if N[1] < 0 { // Quadrant check.
		Ω = 2*math.Pi - Ω
	}

	ω = math.Acos(dot(N, eVec) / (norm(N) * e))
	if eVec[2] < 0 { // Quadrant check
		ω = 2*math.Pi - ω
	}

	ν = math.Acos(dot(eVec, o.R) / (e * norm(o.R)))
	if dot(o.R, o.V) < 0 {
		ν = 2*math.Pi - ν
	}

	return
}

// Astrocodile is an orbit propagator.
// It's a play on words from STK's Atrogrator.
type Astrocodile struct {
	Center    *CelestialObject
	Vehicle   *Spacecraft
	Orbit     *Orbit
	StartDT   *time.Time
	EndDT     *time.Time
	CurrentDT *time.Time
	StopChan  <-chan (bool)
	stepSize  float64 // This duplicates information a bit but is needed for the duration.
}

// NewAstro returns a new Astrocodile instance from the position and velocity vectors.
func NewAstro(c *CelestialObject, s *Spacecraft, R, V []float64, start, end *time.Time) *Astrocodile {
	return &Astrocodile{c, s, NewOrbit(R, V, c.μ), start, end, start, make(chan (bool), 1), 1e-6}
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	integrator.NewRK4(0, a.stepSize, a)
}

// Stop implements the stop call of the integrator.
func (a *Astrocodile) Stop(i uint64) bool {
	// TODO: Add the waiting on the channel and block it.
	// Possibly need an attribute and a listening goroutine.
	// a.CurrentDT.Add(a.stepSize * time.Second)
	return true
}

// GetState returns the state for the integrator.
func (a *Astrocodile) GetState() (s []float64) {
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
	a.Orbit.R[0] = s[0]
	a.Orbit.R[1] = s[1]
	a.Orbit.R[2] = s[2]
	a.Orbit.V[0] = s[3]
	a.Orbit.V[1] = s[4]
	a.Orbit.V[2] = s[5]
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, s []float64) (f []float64) {
	r := mat64.Norm(mat64.NewVector(3, []float64{s[0], s[1], s[2]}), 2)
	f[0] = s[3]
	f[1] = s[4]
	f[2] = s[5]
	f[3] = a.Orbit.μ * s[0] / math.Pow(r, 3)
	f[4] = a.Orbit.μ * s[1] / math.Pow(r, 3)
	f[5] = a.Orbit.μ * s[2] / math.Pow(r, 3)
	return
}
