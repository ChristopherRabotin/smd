package smd

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

const (
	eccentricityε = 5e-5                         // 0.00005
	angleε        = (5e-3 / 360) * (2 * math.Pi) // 0.005 degrees
	distanceε     = 2e1                          // 20 km
	velocityε     = 1e-6                         // in km/s
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	a, e, i, Ω, ω, ν float64
	Origin           CelestialObject // Orbit origin
	cacheHash        float64
	cachedR, cachedV []float64
}

// Energyξ returns the specific mechanical energy ξ.
func (o Orbit) Energyξ() float64 {
	return -o.Origin.μ / (2 * o.a)
}

// Tildeω returns the longitude of periapsis.
func (o Orbit) Tildeω() float64 {
	return math.Mod(o.ω+o.Ω, 2*math.Pi)
}

// TrueLongλ returns the *approximate* true longitude (cf. Vallado page 103).
// NOTE: One should only need this for equatorial orbits.
func (o Orbit) TrueLongλ() float64 {
	return math.Mod(o.ω+o.Ω+o.ν, 2*math.Pi)
}

// ArgLatitudeU returns the argument of latitude.
func (o Orbit) ArgLatitudeU() float64 {
	return math.Mod(o.ν+o.ω, 2*math.Pi)
}

// H returns the orbital angular momentum vector.
func (o Orbit) H() []float64 {
	return cross(o.RV())
}

// HNorm returns the norm of orbital angular momentum.
func (o Orbit) HNorm() float64 {
	return o.RNorm() * o.VNorm() * o.CosΦfpa()
}

// CosΦfpa returns the cosine of the flight path angle.
// WARNING: As per Vallado page 105, *do not* use math.Acos(o.CosΦfpa())
// to get the flight path angle as you'll have a quadran problem. Instead
// use math.Atan2(o.GetSinΦfpa(), o.CosΦfpa()).
func (o Orbit) CosΦfpa() float64 {
	ecosν := o.e * math.Cos(o.ν)
	return (1 + ecosν) / math.Sqrt(1+2*ecosν+math.Pow(o.e, 2))
}

// SinΦfpa returns the cosine of the flight path angle.
// WARNING: As per Vallado page 105, *do not* use math.Asin(o.SinΦfpa())
// to get the flight path angle as you'll have a quadran problem. Instead
// use math.Atan2(o.SinΦfpa(), o.CosΦfpa()).
func (o Orbit) SinΦfpa() float64 {
	sinν, cosν := math.Sincos(o.ν)
	return (o.e * sinν) / math.Sqrt(1+2*o.e*cosν+math.Pow(o.e, 2))
}

// SemiParameter returns the apoapsis.
func (o Orbit) SemiParameter() float64 {
	return o.a * (1 - o.e*o.e)
}

// Apoapsis returns the apoapsis.
func (o Orbit) Apoapsis() float64 {
	return o.a * (1 + o.e)
}

// Periapsis returns the apoapsis.
func (o Orbit) Periapsis() float64 {
	return o.a * (1 - o.e)
}

// SinCosE returns the eccentric anomaly trig functions (sin and cos).
func (o Orbit) SinCosE() (sinE, cosE float64) {
	sinν, cosν := math.Sincos(o.ν)
	denom := 1 + o.e*cosν
	sinE = math.Sqrt(1-o.e*o.e) * sinν / denom
	cosE = (o.e + cosν) / denom
	return
}

// Period returns the period of this orbit.
func (o Orbit) Period() time.Duration {
	// The time package does not trivially handle fractions of a second, so let's
	// compute this in a convoluted way...
	seconds := 2 * math.Pi * math.Sqrt(math.Pow(o.a, 3)/o.Origin.μ)
	duration, _ := time.ParseDuration(fmt.Sprintf("%.6fs", seconds))
	return duration
}

// RV helps with the cache.
func (o *Orbit) RV() ([]float64, []float64) {
	if o.hashValid() {
		return o.cachedR, o.cachedV
	}
	p := o.SemiParameter()
	// Support special orbits.
	ν := o.ν
	ω := o.ω
	Ω := o.Ω
	if o.e < eccentricityε {
		ω = 0
		if o.i < angleε {
			// Circular equatorial
			Ω = 0
			ν = o.TrueLongλ()
		} else {
			// Circular inclined
			ν = o.ArgLatitudeU()
		}
	} else if o.i < angleε {
		Ω = 0
		ω = o.Tildeω()
	}

	R := make([]float64, 3)
	V := make([]float64, 3)
	sinν, cosν := math.Sincos(ν)
	R[0] = p * cosν / (1 + o.e*cosν)
	R[1] = p * sinν / (1 + o.e*cosν)
	R[2] = 0
	R = PQW2ECI(o.i, ω, Ω, R)

	V = make([]float64, 3, 3)
	V[0] = -math.Sqrt(o.Origin.μ/p) * sinν
	V[1] = math.Sqrt(o.Origin.μ/p) * (o.e + cosν)
	V[2] = 0
	V = PQW2ECI(o.i, ω, Ω, V)

	o.cachedR = R
	o.cachedV = V
	o.computeHash()
	return R, V
}

// R returns the radius vector.
func (o Orbit) R() (R []float64) {
	R, _ = o.RV()
	return R
}

// RNorm returns the norm of the radius vector, but without computing the radius vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.R()).
func (o Orbit) RNorm() float64 {
	return o.SemiParameter() / (1 + o.e*math.Cos(o.ν))
}

// V returns the velocity vector.
func (o Orbit) V() (V []float64) {
	_, V = o.RV()
	return V
}

// VNorm returns the norm of the velocity vector, but without computing the velocity vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.GetV()).
func (o Orbit) VNorm() float64 {
	if floats.EqualWithinAbs(o.e, 0, eccentricityε) {
		return math.Sqrt(o.Origin.μ / o.RNorm())
	}
	if floats.EqualWithinAbs(o.e, 1, eccentricityε) {
		return math.Sqrt(2 * o.Origin.μ / o.RNorm())
	}
	return math.Sqrt(2 * (o.Origin.μ/o.RNorm() + o.Energyξ()))
}

// Elements returns the nine orbital elements which work in all types of orbits
func (o *Orbit) Elements() (a, e, i, Ω, ω, ν, λ, tildeω, u float64) {
	a = o.a
	e = o.e
	i = o.i
	Ω = o.Ω
	ω = o.ω
	ν = o.ν
	λ = o.TrueLongλ()
	tildeω = o.Tildeω()
	u = o.ArgLatitudeU()
	return
}

func (o *Orbit) computeHash() {
	o.cacheHash = o.ω + o.ν + o.Ω + o.i + o.e + o.a
}

func (o Orbit) hashValid() bool {
	return o.cacheHash == o.ω+o.ν+o.Ω+o.i+o.e+o.a
}

// String implements the stringer interface (hence the value receiver)
func (o Orbit) String() string {
	if o.e < eccentricityε {
		// Circular orbit
		if o.i > angleε {
			return fmt.Sprintf("a=%.1f e=%.4f i=%.3f Ω=%.3f u=%.3f", o.a, o.e, Rad2deg(o.i), Rad2deg(o.Ω), Rad2deg(o.ArgLatitudeU()))
		}
		// Equatorial
		return fmt.Sprintf("a=%.1f e=%.4f i=%.3f Ω=%.3f λ=%.3f", o.a, o.e, Rad2deg(o.i), Rad2deg(o.Ω), Rad2deg(o.TrueLongλ()))
	}
	return fmt.Sprintf("a=%.1f e=%.4f i=%.3f Ω=%.3f ω=%.3f ν=%.3f", o.a, o.e, Rad2deg(o.i), Rad2deg(o.Ω), Rad2deg(o.ω), Rad2deg(o.ν))
}

// Equals returns whether two orbits are identical with free true anomaly.
// Use StrictlyEquals to also check true anomaly.
func (o Orbit) Equals(o1 Orbit) (bool, error) {
	if !o.Origin.Equals(o1.Origin) {
		return false, errors.New("different origin")
	}

	if !floats.EqualWithinAbs(o.a, o1.a, distanceε) {
		return false, errors.New("semi major axis invalid")
	}
	if !floats.EqualWithinAbs(o.e, o1.e, eccentricityε) {
		return false, errors.New("eccentricity invalid")
	}
	if !floats.EqualWithinAbs(o.i, o1.i, angleε) {
		return false, errors.New("inclination invalid")
	}
	if !floats.EqualWithinAbs(o.Ω, o1.Ω, angleε) {
		return false, errors.New("RAAN invalid")
	}
	if o.e < eccentricityε {
		// Circular orbit
		if o.i > angleε {
			// Inclined
			if !floats.EqualWithinAbs(o.ArgLatitudeU(), o1.ArgLatitudeU(), angleε) {
				return false, errors.New("argument of latitude invalid")
			}
		} else {
			// Equatorial
			if !floats.EqualWithinAbs(o.TrueLongλ(), o1.TrueLongλ(), angleε) {
				return false, errors.New("true longitude invalid")
			}
		}
	} else if !floats.EqualWithinAbs(o.ω, o1.ω, angleε) {
		return false, errors.New("argument of perigee invalid")
	}

	return true, nil
}

// StrictlyEquals returns whether two orbits are identical.
func (o Orbit) StrictlyEquals(o1 Orbit) (bool, error) {
	// Only check for non circular orbits
	if o.e > eccentricityε && !floats.EqualWithinAbs(o.ν, o1.ν, angleε) {
		return false, errors.New("true anomaly invalid")
	}
	return o.Equals(o1)
}

// ToXCentric converts this orbit the provided celestial object centric equivalent.
// Panics if the vehicle is not within the SOI of the object.
// Panics if already in this frame.
func (o *Orbit) ToXCentric(b CelestialObject, dt time.Time) {
	if o.Origin.Name == b.Name {
		panic(fmt.Errorf("already in orbit around %s", b.Name))
	}
	oR := o.R()
	oV := o.V()
	if b.SOI == -1 {
		// Switch to heliocentric
		// Get planet equatorial coordinates.
		rel := o.Origin.HelioOrbit(dt)
		relR := rel.R()
		relV := rel.V()
		// Switch frame origin.
		for i := 0; i < 3; i++ {
			oR[i] += relR[i]
			oV[i] += relV[i]
		}
	} else {
		// Switch to planet centric
		// Get planet ecliptic coordinates.
		rel := b.HelioOrbit(dt)
		relR := rel.R()
		relV := rel.V()
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
// WARNING: Angles must be in degrees not radian.
func NewOrbitFromOE(a, e, i, Ω, ω, ν float64, c CelestialObject) *Orbit {
	// Making an approximation for circular and equatorial orbits.
	if e < eccentricityε {
		e = eccentricityε
	}
	if i < angleε {
		i = angleε
	}
	orbit := Orbit{a, e, Deg2rad(i), Deg2rad(Ω), Deg2rad(ω), Deg2rad(ν), c, 0.0, nil, nil}
	orbit.RV()
	orbit.computeHash()
	return &orbit
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
	if math.IsNaN(ω) {
		ω = 0
	}
	if eVec[2] < 0 {
		ω = 2*math.Pi - ω
	}
	Ω := math.Acos(n[0] / norm(n))
	if n[1] < 0 {
		Ω = 2*math.Pi - Ω
	}
	cosν := dot(eVec, R) / (e * r)
	if abscosν := math.Abs(cosν); abscosν > 1 && floats.EqualWithinAbs(abscosν, 1, 1e-12) {
		// Welcome to the edge case which took about 1.5 hours of my time.
		cosν = sign(cosν) // GTFO NaN!
	}
	ν := math.Acos(cosν)
	if dot(R, V) < 0 {
		ν = 2*math.Pi - ν
	}
	// Fix rounding errors.
	i = math.Mod(i, 2*math.Pi)
	Ω = math.Mod(Ω, 2*math.Pi)
	ω = math.Mod(ω, 2*math.Pi)
	ν = math.Mod(ν, 2*math.Pi)

	orbit := Orbit{a, e, i, Ω, ω, ν, c, 0.0, R, V}
	orbit.computeHash()
	return &orbit
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
