package smd

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

const (
	// Precise ε
	eccentricityε = 5e-5                         // 0.00005
	angleε        = (5e-3 / 360) * (2 * math.Pi) // 0.005 degrees
	distanceε     = 2e1                          // 20 km
	// Coarse ε (for interplanetary flight)
	eccentricityLgε = 1e-2                         // 0.01
	angleLgε        = (5e-1 / 360) * (2 * math.Pi) // 0.5 degrees
	distanceLgε     = 5e2                          // 500 km
	// velocity ε for circular orbit equality and Hohmann
	velocityε = 1e-4 // in km/s
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	rVec, vVec []float64       // Stars with a lowercase to make private
	Origin     CelestialObject // Orbit origin
	// Cache management
	cacheHash, ccha, cche, cchi, cchΩ, cchω, cchν, cchλ, cchtildeω, cchu float64
}

// Energyξ returns the specific mechanical energy ξ.
func (o Orbit) Energyξ() float64 {
	return math.Pow(o.VNorm(), 2)/2 - o.Origin.μ/o.RNorm()
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
	_, e, _, _, _, ν, _, _, _ := o.Elements()
	if e < eccentricityε {
		return 1
	} else if floats.EqualWithinAbs(e, 1, eccentricityε) {
		return math.Cos(ν / 2)
	} else if e > 1 {
		cosh2 := math.Pow((e+math.Cos(ν))/(1+e*math.Cos(ν)), 2)
		return math.Sqrt((e*e - 1) / (e*e*cosh2 - 1))
	}
	ecosν := e * math.Cos(ν)
	return (1 + ecosν) / math.Sqrt(1+2*ecosν+math.Pow(e, 2))
}

// SinΦfpa returns the cosine of the flight path angle.
// WARNING: As per Vallado page 105, *do not* use math.Asin(o.SinΦfpa())
// to get the flight path angle as you'll have a quadran problem. Instead
// use math.Atan2(o.SinΦfpa(), o.CosΦfpa()).
func (o Orbit) SinΦfpa() float64 {
	_, e, _, _, _, ν, _, _, _ := o.Elements()
	if e < eccentricityε {
		return 0
	} else if floats.EqualWithinAbs(e, 1, eccentricityε) {
		return math.Sin(ν / 2)
	} else if e > 1 {
		sinν, cosν := math.Sincos(ν)
		cosh2 := math.Pow((e+cosν)/(1+e*cosν), 2)
		sinh := sinν * math.Sqrt(e*e-1) / (1 + e*cosν)
		return -(e * sinh) / math.Sqrt(e*e*cosh2-1)
	}
	sinν, cosν := math.Sincos(ν)
	return (e * sinν) / math.Sqrt(1+2*e*cosν+math.Pow(e, 2))
}

// SemiParameter returns the apoapsis.
func (o Orbit) SemiParameter() float64 {
	a, e, _, _, _, _, _, _, _ := o.Elements()
	return a * (1 - e*e)
}

// Apoapsis returns the apoapsis.
func (o Orbit) Apoapsis() float64 {
	a, e, _, _, _, _, _, _, _ := o.Elements()
	return a * (1 + e)
}

// Periapsis returns the apoapsis.
func (o Orbit) Periapsis() float64 {
	a, e, _, _, _, _, _, _, _ := o.Elements()
	return a * (1 - e)
}

// SinCosE returns the eccentric anomaly trig functions (sin and cos).
func (o Orbit) SinCosE() (sinE, cosE float64) {
	_, e, _, _, _, ν, _, _, _ := o.Elements()
	sinν, cosν := math.Sincos(ν)
	denom := 1 + e*cosν
	if e > 1 {
		// Hyperbolic orbit
		sinE = math.Sqrt(e*e-1) * sinν / denom
	} else {
		sinE = math.Sqrt(1-e*e) * sinν / denom
	}
	cosE = (e + cosν) / denom
	return
}

// Period returns the period of this orbit.
func (o Orbit) Period() time.Duration {
	// The time package does not trivially handle fractions of a second, so let's
	// compute this in a convoluted way...
	a, _, _, _, _, _, _, _, _ := o.Elements()
	seconds := 2 * math.Pi * math.Sqrt(math.Pow(a, 3)/o.Origin.μ)
	duration, _ := time.ParseDuration(fmt.Sprintf("%.6fs", seconds))
	return duration
}

// RV helps with the cache.
func (o Orbit) RV() ([]float64, []float64) {
	return o.rVec, o.vVec
}

// R returns the radius vector.
func (o Orbit) R() (R []float64) {
	return o.rVec
}

// RNorm returns the norm of the radius vector, but without computing the radius vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.R()).
func (o Orbit) RNorm() float64 {
	return norm(o.rVec)
}

// V returns the velocity vector.
func (o Orbit) V() (V []float64) {
	return o.vVec
}

// VNorm returns the norm of the velocity vector, but without computing the velocity vector.
// If only the norm is needed, it is encouraged to use this function instead of norm(o.GetV()).
func (o Orbit) VNorm() float64 {
	return norm(o.vVec)
}

// Elements returns the nine orbital elements in radians which work for circular and elliptical orbits
func (o *Orbit) Elements() (a, e, i, Ω, ω, ν, λ, tildeω, u float64) {
	if o.hashValid() {
		return o.ccha, o.cche, o.cchi, o.cchΩ, o.cchω, o.cchν, o.cchλ, o.cchtildeω, o.cchu
	}
	// Algorithm from Vallado, 4th edition, page 113 (RV2COE).
	hVec := cross(o.rVec, o.vVec)
	n := cross([]float64{0, 0, 1}, hVec)
	v := norm(o.vVec)
	r := norm(o.rVec)
	ξ := (v*v)/2 - o.Origin.μ/r
	a = -o.Origin.μ / (2 * ξ)
	eVec := make([]float64, 3, 3)
	for i := 0; i < 3; i++ {
		eVec[i] = ((v*v-o.Origin.μ/r)*o.rVec[i] - dot(o.rVec, o.vVec)*o.vVec[i]) / o.Origin.μ
	}
	e = norm(eVec)
	// Prevent nil values for e
	if e < eccentricityε {
		e = eccentricityε
	}
	i = math.Acos(hVec[2] / norm(hVec))
	if i < angleε {
		i = angleε
	}
	ω = math.Acos(dot(n, eVec) / (norm(n) * e))
	if math.IsNaN(ω) {
		ω = 0
	}
	if eVec[2] < 0 {
		ω = 2*math.Pi - ω
	}
	Ω = math.Acos(n[0] / norm(n))
	if math.IsNaN(Ω) {
		Ω = angleε
	}
	if n[1] < 0 {
		Ω = 2*math.Pi - Ω
	}
	cosν := dot(eVec, o.rVec) / (e * r)
	if abscosν := math.Abs(cosν); abscosν > 1 && floats.EqualWithinAbs(abscosν, 1, 1e-12) {
		// Welcome to the edge case which took about 1.5 hours of my time.
		cosν = sign(cosν) // GTFO NaN!
	}
	ν = math.Acos(cosν)
	if math.IsNaN(ν) {
		ν = 0
	}
	if dot(o.rVec, o.vVec) < 0 {
		ν = 2*math.Pi - ν
	}
	// Fix rounding errors.
	i = math.Mod(i, 2*math.Pi)
	Ω = math.Mod(Ω, 2*math.Pi)
	ω = math.Mod(ω, 2*math.Pi)
	ν = math.Mod(ν, 2*math.Pi)
	λ = math.Mod(ω+Ω+ν, 2*math.Pi)
	tildeω = math.Mod(ω+Ω, 2*math.Pi)
	if e < eccentricityε {
		// Circular
		u = math.Acos(dot(n, o.rVec) / (norm(n) * r))
	} else {
		u = math.Mod(ν+ω, 2*math.Pi)
	}
	// Cache values
	o.ccha = a
	o.cche = e
	o.cchi = i
	o.cchΩ = Ω
	o.cchω = ω
	o.cchν = ν
	o.cchλ = λ
	o.cchtildeω = tildeω
	o.cchu = u
	o.computeHash()
	return
}

// MeanAnomaly returns the mean anomaly for hyperbolic orbits only.
func (o Orbit) MeanAnomaly() float64 {
	_, e, _, _, _, _, _, _, _ := o.Elements()
	sinH, cosH := o.SinCosE()
	H := math.Atan2(sinH, cosH)
	return e*math.Sinh(H) - H
}

func (o *Orbit) computeHash() {
	o.cacheHash = 0
	for i := 0; i < 3; i++ {
		o.cacheHash += o.rVec[i] + o.vVec[i]
	}
}

func (o Orbit) hashValid() bool {
	exptdHash := 0.0
	for i := 0; i < 3; i++ {
		exptdHash += o.rVec[i] + o.vVec[i]
	}
	return o.cacheHash == exptdHash
}

// String implements the stringer interface (hence the value receiver)
func (o Orbit) String() string {
	a, e, i, Ω, ω, ν, λ, _, u := o.Elements()
	return fmt.Sprintf("r=%.1f a=%.1f e=%.4f i=%.3f Ω=%.3f ω=%.3f ν=%.3f λ=%.3f u=%.3f", norm(o.rVec), a, e, Rad2deg(i), Rad2deg(Ω), Rad2deg(ω), Rad2deg(ν), Rad2deg(λ), Rad2deg(u))
}

// epsilons returns the epsilons used to determine equality.
func (o Orbit) epsilons() (float64, float64, float64) {
	if o.Origin.Equals(Sun) {
		return distanceLgε, eccentricityLgε, angleLgε
	}
	return distanceε, eccentricityε, angleε
}

// Equals returns whether two orbits are identical with free true anomaly.
// Use StrictlyEquals to also check true anomaly.
func (o Orbit) Equals(o1 Orbit) (bool, error) {
	if !o.Origin.Equals(o1.Origin) {
		return false, errors.New("different origin")
	}
	a, e, i, Ω, ω, _, λ, _, u := o.Elements()
	a1, e1, i1, Ω1, ω1, _, λ1, _, u1 := o1.Elements()
	if !floats.EqualWithinAbs(a, a1, distanceε) {
		return false, errors.New("semi major axis invalid")
	}
	if !floats.EqualWithinAbs(e, e1, eccentricityε) {
		return false, errors.New("eccentricity invalid")
	}
	if !floats.EqualWithinAbs(i, i1, angleε) {
		return false, errors.New("inclination invalid")
	}
	if !floats.EqualWithinAbs(Ω, Ω1, angleε) {
		return false, errors.New("RAAN invalid")
	}
	if e < eccentricityε {
		// Circular orbit
		if i > angleε {
			// Inclined
			if !floats.EqualWithinAbs(u, u1, angleε) {
				return false, errors.New("argument of latitude invalid")
			}
		} else {
			// Equatorial
			if !floats.EqualWithinAbs(λ, λ1, angleε) {
				return false, errors.New("true longitude invalid")
			}
		}
	} else if !floats.EqualWithinAbs(ω, ω1, angleε) {
		return false, errors.New("argument of perigee invalid")
	}
	return true, nil
}

// StrictlyEquals returns whether two orbits are identical.
func (o Orbit) StrictlyEquals(o1 Orbit) (bool, error) {
	// Only check for non circular orbits
	_, e, _, _, _, ν, _, _, _ := o.Elements()
	_, _, _, _, _, ν1, _, _, _ := o1.Elements()
	if floats.EqualWithinAbs(e, 0, 2*eccentricityε) {
		if floats.EqualApprox(o.rVec, o1.rVec, 1) && floats.EqualApprox(o.vVec, o1.vVec, velocityε) {
			return true, nil
		}
		return false, errors.New("vectors not equal")
	} else if e > eccentricityε && !floats.EqualWithinAbs(ν, ν1, angleε) {
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
	config := smdConfig()
	if config.VSOP87 {
		if b.SOI == -1 {
			// Switch to heliocentric
			// Get planet equatorial coordinates.
			rel := o.Origin.HelioOrbit(dt)
			relR := rel.R()
			relV := rel.V()
			// Switch frame origin.
			for i := 0; i < 3; i++ {
				o.rVec[i] += relR[i]
				o.vVec[i] += relV[i]
			}
		} else {
			// Switch to planet centric
			// Get planet ecliptic coordinates.
			rel := b.HelioOrbit(dt)
			relR := rel.R()
			relV := rel.V()
			// Update frame origin.
			for i := 0; i < 3; i++ {
				o.rVec[i] -= relR[i]
				o.vVec[i] -= relV[i]
			}
		}
	} else if config.SPICE {
		// Using SPICE for the conversion.
		state := make([]float64, 6)
		for i := 0; i < 3; i++ {
			state[i] = o.rVec[i]
			state[i+3] = o.vVec[i]
		}
		toFrame := "IAU_" + b.Name
		if b.Equals(Sun) {
			toFrame = "ECLIPJ2000"
		}
		fromFrame := "IAU_" + o.Origin.Name
		if o.Origin.Equals(Sun) {
			fromFrame = "ECLIPJ2000"
		}
		pstate := config.ChgFrame(toFrame, fromFrame, dt, state)
		o.rVec = pstate.R
		o.vVec = pstate.V
	}
	o.Origin = b // Don't forget to switch origin
}

// NewOrbitFromOE creates an orbit from the orbital elements.
// WARNING: Angles must be in degrees not radians.
func NewOrbitFromOE(a, e, i, Ω, ω, ν float64, c CelestialObject) *Orbit {
	// Convert angles to radians
	i = i * deg2rad
	Ω = Ω * deg2rad
	ω = ω * deg2rad
	ν = ν * deg2rad

	// Algorithm from Vallado, 4th edition, page 118 (COE2RV).
	if e < eccentricityε {
		// Circular...
		if i < angleε {
			// ... equatorial
			Ω = 0
			ω = 0
			ν = math.Mod(ω+Ω+ν, 2*math.Pi)
		} else {
			// ... inclined
			ω = 0
			ν = math.Mod(ν+ω, 2*math.Pi)
		}
	} else if i < angleε {
		// Elliptical equatorial
		Ω = 0
		ω = math.Mod(ω+Ω, 2*math.Pi)
	}
	p := a * (1 - e*e)
	if floats.EqualWithinAbs(e, 1, eccentricityε) || e > 1 {
		panic("[ERROR] should initialize parabolic or hyperbolic orbits with R, V")
	}
	μOp := math.Sqrt(c.μ / p)
	sinν, cosν := math.Sincos(ν)
	rPQW := []float64{p * cosν / (1 + e*cosν), p * sinν / (1 + e*cosν), 0}
	vPQW := []float64{-μOp * sinν, μOp * (e + cosν), 0}
	rIJK := Rot313Vec(-ω, -i, -Ω, rPQW)
	vIJK := Rot313Vec(-ω, -i, -Ω, vPQW)
	orbit := Orbit{rIJK, vIJK, c, a, e, i, Ω, ω, ν, 0, 0, 0, 0.0}
	orbit.Elements()
	return &orbit
}

// NewOrbitFromRV returns orbital elements from the R and V vectors. Needed for prop
func NewOrbitFromRV(R, V []float64, c CelestialObject) *Orbit {
	orbit := Orbit{R, V, c, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0}
	orbit.Elements() // Compute the OEs and the cache hash
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
