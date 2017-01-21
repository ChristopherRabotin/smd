package smd

import (
	"fmt"
	"math"
	"time"

	"github.com/soniakeys/meeus/julian"
	"github.com/soniakeys/meeus/planetposition"
)

const (
	// AU is one astronomical unit in kilometers.
	AU = 149598000
)

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	Name   string
	Radius float64
	a      float64
	μ      float64
	tilt   float64 // Axial tilt
	incl   float64 // Ecliptic inclination
	SOI    float64 // With respect to the Sun
	J2     float64
	PP     *planetposition.V87Planet
}

// String implements the Stringer interface.
func (c CelestialObject) String() string {
	return c.Name + " body"
}

// Equals returns whether the provided celestial object is the same.
func (c *CelestialObject) Equals(b CelestialObject) bool {
	return c.Name == b.Name && c.Radius == b.Radius && c.a == b.a && c.μ == b.μ && c.SOI == b.SOI && c.J2 == b.J2
}

// HelioOrbit returns the heliocentric position and velocity of this planet at a given time in equatorial coordinates.
// Note that the whole file is loaded. In fact, if we don't, then whoever is the first to call this function will
// set the Epoch at which the ephemeris are available, and that sucks.
func (c *CelestialObject) HelioOrbit(dt time.Time) Orbit {
	if c.Name == "Sun" {
		return *NewOrbitFromRV([]float64{0, 0, 0}, []float64{0, 0, 0}, *c)
	}
	if c.PP == nil {
		// Load the planet.
		var vsopPosition int
		switch c.Name {
		case "Venus":
			vsopPosition = 2
			break
		case "Earth":
			vsopPosition = 3
			break
		case "Mars":
			vsopPosition = 4
			break
		default:
			panic(fmt.Errorf("unknown object: %s", c.Name))
		}
		planet, err := planetposition.LoadPlanet(vsopPosition - 1)
		if err != nil {
			panic(fmt.Errorf("could not load planet number %d: %s", vsopPosition, err))
		}
		c.PP = planet
	}
	l, b, r := c.PP.Position2000(julian.TimeToJD(dt))
	r *= AU
	v := math.Sqrt(2*Sun.μ/r - Sun.μ/c.a)
	// Get the Cartesian coordinates from L,B,R.
	R, V := make([]float64, 3), make([]float64, 3)
	sB, cB := math.Sincos(b)
	sL, cL := math.Sincos(l)
	R[0] = r * cB * cL
	R[1] = r * cB * sL
	R[2] = r * sB
	// Let's find the direction of the velocity vector.
	vDir := cross(R, []float64{0, 0, -1})
	for i := 0; i < 3; i++ {
		V[i] = v * vDir[i] / norm(vDir)
	}
	// Correct axial tilt
	R = MxV33(R1(Deg2rad(-c.tilt)), R)
	V = MxV33(R1(Deg2rad(-c.tilt)), V)

	// Correct ecliptic inclination
	R = MxV33(R1(Deg2rad(c.incl)), R)
	V = MxV33(R1(Deg2rad(c.incl)), V)

	return *NewOrbitFromRV(R, V, Sun)
}

/* Definitions */

// Sun is our closest star.
var Sun = CelestialObject{"Sun", 695700, -1, 1.32712440018 * 1e11, 0.0, 0.0, -1, -1, nil}

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 149598023, 3.986004415 * 1e5, 23.4, 0.00005, 924645.0, 0.0010826269, nil}

// Mars is the vacation place.
var Mars = CelestialObject{"Mars", 3397.2, 227939186, 4.305 * 1e4, 25.19, 1.85, 576000, 0.001964, nil}
