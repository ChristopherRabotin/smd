package dynamics

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	a, e, i, Ω, ω, ν float64
	Origin           CelestialObject // Orbit orgin
}

// Energy returns the energy ξ of this orbit.
func (o *Orbit) Energy() float64 {
	return -o.Origin.μ / (2 * o.a)
}

// GetTildeω returns the true longitude of perigee.
func (o *Orbit) GetTildeω() float64 {
	return o.ω + o.Ω
}

// Getλtrue returns the true longitude.
func (o *Orbit) Getλtrue() float64 {
	return o.ω + o.Ω + o.ν
}

// GetU returns the argument of latitude.
func (o *Orbit) GetU() float64 {
	return o.ν + o.ω
}

// GetH returns the orbital angular momentum.
func (o *Orbit) GetH() float64 {
	return norm(cross(o.GetR(), o.GetV()))
}

// GetE returns the eccentricty vector and norm.
func (o *Orbit) GetE() (eVec []float64, e float64) {
	return []float64{0, 0, 0}, o.e
}

// Getν returns the true anomaly.
func (o *Orbit) Getν() (ν float64) {
	return o.ν
}

// GetA returns the semi major axis a.
func (o *Orbit) GetA() (a float64) {
	return o.a
}

// GetSemiParameter returns the apoapsis.
func (o *Orbit) GetSemiParameter() (r float64) {
	r = o.a * (1 - o.e*o.e)
	return
}

// GetApoapsis returns the apoapsis.
func (o *Orbit) GetApoapsis() (r float64) {
	r = o.a * (1 + o.e)
	return
}

// GetPeriapsis returns the apoapsis.
func (o *Orbit) GetPeriapsis() (r float64) {
	r = o.a * (1 - o.e)
	return
}

// GetSinCosE returns the eccentric anomaly trig functions (sin and cos).
func (o *Orbit) GetSinCosE() (sinE, cosE float64) {
	sinν, cosν := math.Sincos(o.ν)
	denom := 1 + o.e*cosν
	sinE = math.Sqrt(1-o.e*o.e) * sinν / denom
	cosE = (o.e + cosν) / denom
	return
}

// GetR returns the radius vector.
func (o *Orbit) GetR() (R []float64) {
	R = make([]float64, 3, 3)
	p := o.GetSemiParameter()
	// Support special orbits.
	ν := o.ν
	ω := o.ω
	Ω := o.Ω
	if o.e < 1e-6 {
		ω = 0
		if o.i < 1e-6 {
			// Circular equatorial
			Ω = 0
			ν = o.Getλtrue()
		} else {
			// Circular inclined
			ν = o.GetU()
		}
	} else if o.i < 1e-6 {
		Ω = 0
		ω = o.GetTildeω()
	}

	sinν, cosν := math.Sincos(ν)
	R[0] = p * cosν / (1 + o.e*cosν)
	R[1] = p * sinν / (1 + o.e*cosν)
	R[2] = 0
	R = PQW2ECI(o.i, ω, Ω, R)
	return
}

// GetV returns the velocity vector.
func (o *Orbit) GetV() (V []float64) {
	V = make([]float64, 3, 3)
	p := o.GetSemiParameter()
	sinν, cosν := math.Sincos(o.ν)
	V[0] = -math.Sqrt(o.Origin.μ/p) * sinν
	V[1] = math.Sqrt(o.Origin.μ/p) * (o.e + cosν)
	V[2] = 0
	V = PQW2ECI(o.i, o.ω, o.Ω, V)
	return
}

// String implements the stringer interface.
func (o *Orbit) String() string {
	return fmt.Sprintf("a=%.3f e=%.3f i=%.3f ω=%.3f Ω=%.3f ν=%.3f", o.a, o.e, Rad2deg(o.i), Rad2deg(o.ω), Rad2deg(o.Ω), Rad2deg(o.ν))
}

// Equals returns whether two orbits are identical.
// WARNING: Does not check the true anomaly.
func (o *Orbit) Equals(o1 Orbit) (bool, error) {
	if !o.Origin.Equals(o1.Origin) {
		return false, errors.New("different origin")
	}
	if floats.EqualWithinAbs(o.a, o1.a, 10) {
		return false, errors.New("semi major axis invalid")
	}
	if floats.EqualWithinAbs(o.e, o1.e, 1e-2) {
		return false, errors.New("eccentricity invalid")
	}
	if floats.EqualWithinAbs(o.i, o1.i, 1e-2) {
		return false, errors.New("inclination invalid")
	}
	if floats.EqualWithinAbs(o.Ω, o1.Ω, 1e-2) {
		return false, errors.New("RAAN invalid")
	}
	if floats.EqualWithinAbs(o.ω, o1.ω, 1e-2) {
		return false, errors.New("argument of perigee invalid")
	}
	return true, nil
}

// ToXCentric converts this orbit the provided celestial object centric equivalent.
// Panics if the vehicle is not within the SOI of the object.
// Panics if already in this frame.
func (o *Orbit) ToXCentric(b CelestialObject, dt time.Time) {
	if o.Origin.Name == b.Name {
		panic(fmt.Errorf("already in orbit around %s", b.Name))
	}
	oR := o.GetR()
	oV := o.GetV()
	if b.SOI == -1 {
		// Switch to heliocentric
		// Get planet equatorial coordinates.
		rel := o.Origin.HelioOrbit(dt)
		relR := rel.GetR()
		relV := rel.GetV()
		// Switch frame origin.
		for i := 0; i < 3; i++ {
			oR[i] += relR[i]
			oV[i] += relV[i]
		}
	} else {
		// Switch to planet centric
		// Get planet ecliptic coordinates.
		rel := b.HelioOrbit(dt)
		relR := rel.GetR()
		relV := rel.GetV()
		// Update frame origin.
		for i := 0; i < 3; i++ {
			oR[i] -= relR[i]
			oV[i] -= relV[i]
		}
	}
	newOrbit := NewOrbitFromRV(oR, oV, b)
	o.a = newOrbit.a
	o.e = newOrbit.e
	o.i = newOrbit.i
	o.Ω = newOrbit.Ω
	o.ν = newOrbit.ν
	o.ω = newOrbit.ω
	o.Origin = b // Don't forget to switch origin
}

// NewOrbitFromOE creates an orbit from the orbital elements.
func NewOrbitFromOE(a, e, i, Ω, ω, ν float64, c CelestialObject) *Orbit {
	return &Orbit{a, e, i, Ω, ω, ν, c}
}

// NewOrbitFromRV returns orbital elements from the R and V vectors. Needed for prop
func NewOrbitFromRV(R, V []float64, c CelestialObject) *Orbit {
	// From Vallado's RV2COE, page 113
	hVec := cross(R, V)
	n := cross([]float64{0, 0, 1}, hVec)
	v := norm(V)
	r := norm(R)
	ξ := (v*v)/2 - c.μ/r
	a := -c.μ / (2 * ξ)
	eVec := make([]float64, 3, 3)
	for i := 0; i < 3; i++ {
		eVec[i] = ((v*v-c.μ/r)*R[i] - dot(R, V)*V[i]) / c.μ
	}
	e := norm(eVec)
	if e >= 1 {
		fmt.Println("[warning] parabolic and hyperpolic orbits not fully supported")
	}
	i := math.Acos(hVec[2] / norm(hVec))
	ω := math.Acos(dot(n, eVec) / (norm(n) * e))
	if eVec[2] < 0 {
		ω = 2*math.Pi - ω
	}
	Ω := math.Acos(n[0] / norm(n))
	if n[1] < 0 {
		Ω = 2*math.Pi - Ω
	}
	ν := math.Acos(dot(eVec, R) / (e * r))
	if dot(R, V) < 0 {
		ν = 2*math.Pi - ν
	}
	return &Orbit{a, e, i, ω, Ω, ν, c}
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
