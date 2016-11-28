package dynamics

import (
	"fmt"
	"math"
	"time"
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	R      []float64       // Radius vector
	V      []float64       // Velocity vector
	Origin CelestialObject // Orbit orgin
}

// Energy returns the energy ξ of this orbit.
func (o *Orbit) Energy() float64 {
	return norm(o.V)*norm(o.V)/2 - o.Origin.μ/norm(o.R)
}

// GetE returns the eccentricty vector and norm.
func (o *Orbit) GetE() (eVec []float64, e float64) {
	eVec = make([]float64, 3)
	for j := 0; j < 3; j++ {
		eVec[j] = ((math.Pow(norm(o.V), 2)-o.Origin.μ/norm(o.R))*o.R[j] - dot(o.R, o.V)*o.V[j]) / o.Origin.μ
	}
	e = norm(eVec) // Eccentricity
	return
}

// Getν returns the true anomaly.
func (o *Orbit) Getν() (ν float64) {
	eVec, e := o.GetE()
	ν = math.Acos(dot(eVec, o.R) / (e * norm(o.R)))
	if dot(o.R, o.V) < 0 {
		ν = 2*math.Pi - ν
	}
	return
}

// GetA returns the semi major axis a.
func (o *Orbit) GetA() (a float64) {
	a = -o.Origin.μ / (2 * (0.5*dot(o.V, o.V) - o.Origin.μ/norm(o.R)))
	return
}

// GetI returns the inclination i.
func (o *Orbit) GetI() (i float64) {
	h := cross(o.R, o.V)
	i = math.Acos(h[2] / norm(h))
	return
}

// GetΩ returns the RAAN Ω.
func (o *Orbit) GetΩ() (Ω float64) {
	h := cross(o.R, o.V)
	N := []float64{-h[1], h[0], 0}

	Ω = math.Acos(N[0] / norm(N))
	if N[1] < 0 { // Quadrant check.
		Ω = 2*math.Pi - Ω
	}
	return
}

// Getω returns the argument of periapsis.
func (o *Orbit) Getω() (ω float64) {
	h := cross(o.R, o.V)
	N := []float64{-h[1], h[0], 0}
	eVec, e := o.GetE()
	ω = math.Acos(dot(N, eVec) / (norm(N) * e))
	if eVec[2] < 0 { // Quadrant check
		ω = 2*math.Pi - ω
	}
	return
}

// GetΦ returns the flight path angle with the correct quadrant.
func (o *Orbit) GetΦ() (Φ float64) {
	_, e := o.GetE()
	sinν, cosν := math.Sincos(o.Getν())
	sinΦ := (e * sinν) / math.Sqrt(1+2*e*cosν+e*e)
	cosΦ := (1 + e*cosν) / math.Sqrt(1+2*e*cosν+e*e)
	Φ = math.Atan2(sinΦ, cosΦ)
	return
}

// OrbitalElements returns the orbital elements of this orbit.
func (o *Orbit) OrbitalElements() (a, e, i, ω, Ω, ν float64) {
	_, e = o.GetE()
	a = o.GetA()
	i = o.GetI()
	Ω = o.GetΩ()
	ω = o.Getω()
	ν = o.Getν()
	return
}

// String implements the stringer interface.
func (o *Orbit) String() string {
	a, e, i, ω, Ω, ν := o.OrbitalElements()
	return fmt.Sprintf("a=%0.5f e=%0.5f i=%0.5f ω=%0.5f Ω=%0.5f ν=%0.5f", a, e, i, ω, Ω, ν)
}

// ToXCentric converts this orbit the provided celestial object centric equivalent.
// Panics if the vehicle is not within the SOI of the object.
// Panics if already in this frame.
func (o *Orbit) ToXCentric(b CelestialObject, dt time.Time) {
	if o.Origin.Name == b.Name {
		panic(fmt.Errorf("already in orbit around %s", b.Name))
	}
	if b.SOI == -1 {
		// Switch to heliocentric
		// Get planet equatorial coordinates.
		rel := o.Origin.HelioOrbit(dt)
		// Switch frame origin.
		for i := 0; i < 3; i++ {
			o.R[i] += rel.R[i]
			o.V[i] += rel.V[i]
		}
	} else {
		// Switch to planet centric
		// Get planet ecliptic coordinates.
		rel := b.HelioOrbit(dt)
		// Update frame origin.
		for i := 0; i < 3; i++ {
			o.R[i] -= rel.R[i]
			o.V[i] -= rel.V[i]
		}
	}
	o.Origin = b // Don't forget to switch origin
}

// NewOrbitFromOE creates an orbit from the orbital elements.
func NewOrbitFromOE(a, e, i, ω, Ω, ν float64, c CelestialObject) *Orbit {
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
	return &Orbit{R, V, c}
}

// NewOrbit returns orbital elements from the R and V vectors. Needed for prop
func NewOrbit(R, V []float64, c CelestialObject) *Orbit {
	return &Orbit{R, V, c}
}

// Helper functions go here.

// Radii2ae returns the semi major axis and the eccentricty from the radii.
func Radii2ae(rA, rP float64) (a, e float64) {
	if rA < rP {
		panic("periapsis cannot be greater than apoapsis")
	}
	a = (rP + rA) / 2
	e = (rA - rP) / (rA + rP)
	return
}
