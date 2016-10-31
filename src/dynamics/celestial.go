package dynamics

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
	SOI    float64 // With respect to the Sun
	J2     float64
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.Name)
}

// Equals returns whether the provided celestial object is the same.
func (c *CelestialObject) Equals(b CelestialObject) bool {
	return c.Name == b.Name && c.Radius == b.Radius && c.a == b.a && c.μ == b.μ && c.SOI == b.SOI && c.J2 == b.J2
}

// HelioOrbit returns the heliocentric position and velocity of this planet at a given time in equatorial coordinates.
func (c *CelestialObject) HelioOrbit(dt time.Time) ([]float64, []float64) {
	var vsopPosition int
	switch c.Name {
	case "Sun":
		return []float64{0, 0, 0}, []float64{0, 0, 0}
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
	// Load planet, note that planetposition starts counting at ZERO!
	if planet, err := planetposition.LoadPlanet(vsopPosition - 1); err != nil {
		panic(fmt.Errorf("could not load planet number %d: %s", vsopPosition, err))
	} else {
		l, b, r := planet.Position2000(julian.TimeToJD(dt))
		r *= AU
		v := math.Sqrt(2*Sun.μ/r - Sun.μ/c.a)
		// Get the Cartesian coordinates from L,B,R.
		rEcliptic, vEcliptic := make([]float64, 3), make([]float64, 3)
		sB, cB := math.Sincos(b)
		sL, cL := math.Sincos(l)
		rEcliptic[0] = r * cB * cL
		rEcliptic[1] = r * cB * sL
		rEcliptic[2] = r * sB
		vEcliptic[1] = v * cB * cL
		vEcliptic[0] = v * cB * sL * -1
		vEcliptic[2] = v * sB
		/*
					--> velocity
					got = [18.37369994215764 -0.0005404756532697506 23.715956543653625]
					got = [23.715956533393097 18.37369994215764 -0.0008824910035900467]
			    exp = [-18.735531582133625 23.452304089603338 -0.0012486895377029493]
			    exp = [-18.735531582133625 23.452304089603338 -0.0012486895377029493]
		*/
		return rEcliptic, vEcliptic
	}

}

/* Definitions */

// Sun is our closest star.
var Sun = CelestialObject{"Sun", 695700, -1, 1.32712440018 * 1e11, 0.0, -1, -1}

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 149598023, 3.986004415 * 1e5, 23.4, 924645.0, 0.0010826269}

// Mars is the vacation place.
var Mars = CelestialObject{"Mars", 3397.2, 227939186, 4.305 * 1e4, 25.19, 576000, 0.001964}
