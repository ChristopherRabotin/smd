package smd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/soniakeys/meeus/julian"
	"github.com/soniakeys/meeus/planetposition"
	"github.com/soniakeys/meeus/pluto"
)

const (
	// AU is one astronomical unit in kilometers.
	AU = 1.49597870700e8
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
	J3     float64
	J4     float64
	PP     *planetposition.V87Planet
}

// GM returns μ (which is unexported because it's a lowercase letter)
func (c CelestialObject) GM() float64 {
	return c.μ
}

// J returns the perturbing J_n factor for the provided n.
// Currently only J2 and J3 are supported.
func (c CelestialObject) J(n uint8) float64 {
	switch n {
	case 2:
		return c.J2
	case 3:
		return c.J3
	case 4:
		return c.J4
	default:
		return 0.0
	}
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
	if smdConfig().VSOP87 {
		if c.Name == "Pluto" {
			// Special case in Sonia Keys' Meeus
			l, b, r := pluto.Heliocentric(julian.TimeToJD(dt))
			r *= AU
			v := math.Sqrt(2*Sun.μ/r - Sun.μ/c.a)
			// Get the Cartesian coordinates from L,B,R.
			R, V := make([]float64, 3), make([]float64, 3)
			sB, cB := math.Sincos(b.Rad())
			sL, cL := math.Sincos(l.Rad())
			R[0] = r * cB * cL
			R[1] = r * cB * sL
			R[2] = r * sB
			// Let's find the direction of the velocity vector.
			vDir := Cross(R, []float64{0, 0, -1})
			for i := 0; i < 3; i++ {
				V[i] = v * vDir[i] / Norm(vDir)
			}
			return *NewOrbitFromRV(R, V, Sun)
		}
		if c.PP == nil {
			// Load the planet.
			var vsopPosition int
			switch c.Name {
			case "Venus":
				vsopPosition = 2
			case "Earth":
				vsopPosition = 3
			case "Mars":
				vsopPosition = 4
			case "Jupiter":
				vsopPosition = 5
			default:
				panic(fmt.Errorf("unknown object: %s", c.Name))
			}
			planet, err := planetposition.LoadPlanetPath(vsopPosition-1, smdConfig().VSOP87Dir)
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
		sB, cB := math.Sincos(b.Rad())
		sL, cL := math.Sincos(l.Rad())
		R[0] = r * cB * cL
		R[1] = r * cB * sL
		R[2] = r * sB
		// Let's find the direction of the velocity vector.
		vDir := Cross(R, []float64{0, 0, -1})
		for i := 0; i < 3; i++ {
			V[i] = v * vDir[i] / Norm(vDir)
		}
		return *NewOrbitFromRV(R, V, Sun)
	}
	pstate := config.HelioState(c.Name, dt)
	R := pstate.R
	V := pstate.V
	return *NewOrbitFromRV(R, V, Sun)
}

// CelestialObjectFromString returns the object from its name
func CelestialObjectFromString(name string) (CelestialObject, error) {
	switch strings.ToLower(name) {
	case "earth":
		return Earth, nil
	case "venus":
		return Venus, nil
	case "mars":
		return Mars, nil
	case "jupiter":
		return Jupiter, nil
	case "saturn":
		return Saturn, nil
	case "uranus":
		return Uranus, nil
	case "pluto":
		return Pluto, nil
	default:
		return CelestialObject{}, fmt.Errorf("undefined planet '%s'", name)
	}
}

/* Definitions */

// Sun is our closest star.
var Sun = CelestialObject{"Sun", 695700, -1, 1.32712440017987e11, 0.0, 0.0, -1, 0, 0, 0, nil}

// Venus is poisonous.
var Venus = CelestialObject{"Venus", 6051.8, 108208601, 3.24858599e5, 117.36, 3.39458, 0.616e6, 0.000027, 0, 0, nil}

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 149598023, 3.98600433e5, 23.4, 0.00005, 924645.0, 1082.6269e-6, -2.5324e-6, -1.6204e-6, nil}

// Mars is the vacation place.
var Mars = CelestialObject{"Mars", 3396.19, 227939282.5616, 4.28283100e4, 25.19, 1.85, 576000, 1964e-6, 36e-6, -18e-6, nil}

// Jupiter is big.
var Jupiter = CelestialObject{"Jupiter", 71492.0, 778298361, 1.266865361e8, 3.13, 1.30326966, 48.2e6, 0.01475, 0, -0.00058, nil}

// Saturn floats and that's really cool.
// TODO: SOI
var Saturn = CelestialObject{"Saturn", 60268.0, 1429394133, 3.7931208e7, 0.93, 2.485, 0, 0.01645, 0, -0.001, nil}

// Uranus is no joke.
// TODO: SOI
var Uranus = CelestialObject{"Uranus", 25559.0, 2875038615, 5.7939513e6, 1.02, 0.773, 0, 0.012, 0, 0, nil}

// Pluto is not a planet and had that down ranking coming. It should have stayed in its lane.
// WARNING: Pluto SOI is not defined.
var Pluto = CelestialObject{"Pluto", 1151.0, 5915799000, 9. * 1e2, 118.0, 17.14216667, 1, 0, 0, 0, nil}
