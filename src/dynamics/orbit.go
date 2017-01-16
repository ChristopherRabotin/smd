package dynamics

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

const (
	eccentricityε = 1e-2
	angleε        = (1e-2 / 360) * (2 * math.Pi) // Within 0.01 degrees.
	distanceε     = 5e1                          // 50 km
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	a, e, i, Ω, ω, ν float64
	Origin           CelestialObject // Orbit orgin
	cacheHash        float64
	cachedR, cachedV []float64
}

// Getξ returns the specific mechanical energy ξ.
func (o *Orbit) Getξ() float64 {
	return -o.Origin.μ / (2 * o.a)
}

// GetTildeω returns the longitude of periapsis.
func (o *Orbit) GetTildeω() float64 {
	return o.ω + o.Ω
}

// Getλtrue returns the *approximate* true longitude.
func (o *Orbit) Getλtrue() float64 {
	if o.e > eccentricityε || o.i > angleε {
		panic("Getλtrue only supports circular equatorial orbits")
	}
	// This is an approximation as per Vallado page 103.
	return o.ω + o.Ω + o.ν
}

// GetU returns the argument of latitude.
func (o *Orbit) GetU() float64 {
	return o.ν + o.ω
}

// GetH returns the orbital angular momentum vector.
func (o *Orbit) GetH() []float64 {
	return cross(o.GetRV())
}

// GetHNorm returns the norm of orbital angular momentum.
func (o *Orbit) GetHNorm() float64 {
	return o.GetRNorm() * o.GetVNorm() * o.GetCosΦfpa()
}

// GetCosΦfpa returns the cosine of the flight path angle.
// WARNING: As per Vallado page 105, *do not* use math.Acos(o.GetCosΦfpa())
// to get the flight path angle as you'll have a quadran problem. Instead
// use math.Atan2(o.GetSinΦfpa(), o.GetCosΦfpa()).
func (o *Orbit) GetCosΦfpa() float64 {
	ecosν := o.e * math.Cos(o.ν)
	return (1 + ecosν) / math.Sqrt(1+2*ecosν+math.Pow(o.e, 2))
}

// GetSinΦfpa returns the cosine of the flight path angle.
// WARNING: As per Vallado page 105, *do not* use math.Asin(o.GetSinΦfpa())
// to get the flight path angle as you'll have a quadran problem. Instead
// use math.Atan2(o.GetSinΦfpa(), o.GetCosΦfpa()).
func (o *Orbit) GetSinΦfpa() float64 {
	sinν, cosν := math.Sincos(o.ν)
	return (o.e * sinν) / math.Sqrt(1+2*o.e*cosν+math.Pow(o.e, 2))
}

// GetSemiParameter returns the apoapsis.
func (o *Orbit) GetSemiParameter() float64 {
	return o.a * (1 - o.e*o.e)
}

// GetApoapsis returns the apoapsis.
func (o *Orbit) GetApoapsis() float64 {
	return o.a * (1 + o.e)
}

// GetPeriapsis returns the apoapsis.
func (o *Orbit) GetPeriapsis() float64 {
	return o.a * (1 - o.e)
}

// GetSinCosE returns the eccentric anomaly trig functions (sin and cos).
func (o *Orbit) GetSinCosE() (sinE, cosE float64) {
	sinν, cosν := math.Sincos(o.ν)
	denom := 1 + o.e*cosν
	sinE = math.Sqrt(1-o.e*o.e) * sinν / denom
	cosE = (o.e + cosν) / denom
	return
}

// GetRV helps with the cache.
func (o *Orbit) GetRV() ([]float64, []float64) {
	if o.hashValid() {
		return o.cachedR, o.cachedV
	}
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

// GetR returns the radius vector.
func (o *Orbit) GetR() (R []float64) {
	R, _ = o.GetRV()
	return R
}

// GetRNorm returns the norm of the radius vector, but without computing the radius vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.GetR()).
func (o *Orbit) GetRNorm() float64 {
	return o.GetSemiParameter() / (1 + o.e*math.Cos(o.ν))
}

// GetV returns the velocity vector.
func (o *Orbit) GetV() (V []float64) {
	_, V = o.GetRV()
	return V
}

// GetVNorm returns the norm of the velocity vector, but without computing the velocity vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.GetV()).
func (o *Orbit) GetVNorm() float64 {
	if floats.EqualWithinAbs(o.e, 0, eccentricityε) {
		return math.Sqrt(o.Origin.μ / o.GetRNorm())
	}
	if floats.EqualWithinAbs(o.e, 1, eccentricityε) {
		return math.Sqrt(2 * o.Origin.μ / o.GetRNorm())
	}
	return math.Sqrt(2 * (o.Origin.μ/o.GetRNorm() + o.Getξ()))
}

func (o *Orbit) computeHash() {
	o.cacheHash = o.ω + o.ν + o.Ω + o.i + o.e + o.a
}

func (o *Orbit) hashValid() bool {
	return o.cacheHash == o.ω+o.ν+o.Ω+o.i+o.e+o.a
}

// String implements the stringer interface (hence the value receiver)
func (o Orbit) String() string {
	return fmt.Sprintf("a=%.3f e=%.3f i=%.3f ω=%.3f Ω=%.3f ν=%.3f", o.a, o.e, Rad2deg(o.i), Rad2deg(o.ω), Rad2deg(o.Ω), Rad2deg(o.ν))
}

// Equals returns whether two orbits are identical with free true anomaly.
// Use StrictlyEquals to also check true anomaly.
func (o *Orbit) Equals(o1 Orbit) (bool, error) {
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
	if !floats.EqualWithinAbs(o.ω, o1.ω, angleε) {
		return false, errors.New("argument of perigee invalid")
	}
	return true, nil
}

// StrictlyEquals returns whether two orbits are identical.
func (o *Orbit) StrictlyEquals(o1 Orbit) (bool, error) {
	if !floats.EqualWithinAbs(o.ν, o1.ν, angleε) {
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
// WARNING: Angles must be in degrees not radian.
func NewOrbitFromOE(a, e, i, Ω, ω, ν float64, c CelestialObject) *Orbit {
	orbit := Orbit{a, e, Deg2rad(i), Deg2rad(Ω), Deg2rad(ω), Deg2rad(ν), c, 0.0, nil, nil}
	orbit.GetRV()
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
