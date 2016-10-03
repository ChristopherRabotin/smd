package dynamics

import (
	"fmt"
	"math"
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	R []float64 // Radius vector
	V []float64 // Velocity vector
	μ float64   // Gravitational constant of the center of orbit.
}

// NewOrbitFromOE creates an orbit from the orbital elements.
func NewOrbitFromOE(a, e, i, ω, Ω, ν float64, c *CelestialObject) *Orbit {
	// Check for edge cases which are not supported.
	if ν < 1e-10 {
		panic("ν ~= 0 is not supported")
	}
	if e < 0 || e > 1 {
		panic("only circular and elliptical orbits supported")
	}
	μ := c.μ
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
func NewOrbit(R, V []float64, c *CelestialObject) *Orbit {
	return &Orbit{R, V, c.μ}
}

// GetOE returns the orbital elements of this orbit.
func (o *Orbit) GetOE() (a, e, i, ω, Ω, ν float64) {
	h := cross(o.R, o.V)

	N := []float64{-h[1], h[0], 0}

	eVec := make([]float64, 3)
	for j := 0; j < 3; j++ {
		eVec[j] = ((math.Pow(norm(o.V), 2)-o.μ/norm(o.R))*o.R[j] - dot(o.R, o.V)*o.V[j]) / o.μ
	}
	e = norm(eVec) // Eccentricity
	// We suppose the orbit is NOT parabolic.
	a = -o.μ / (2 * (0.5*dot(o.V, o.V) - o.μ/norm(o.R)))
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

// String implements the stringer interface.
func (o *Orbit) String() string {
	a, e, i, ω, Ω, ν := o.GetOE()
	return fmt.Sprintf("a=%0.5f e=%0.5f i=%0.5f ω=%0.5f Ω=%0.5f ν=%0.5f", a, e, i, ω, Ω, ν)
}
