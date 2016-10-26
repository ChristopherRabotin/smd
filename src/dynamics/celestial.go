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
	SOI    float64 // With respect to the Sun
	J2     float64
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.Name)
}

// HelioOrbit returns the heliocentric position and velocity of this planet at a given time.
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
	// Load planet.
	if planet, err := planetposition.LoadPlanet(vsopPosition); err != nil {
		panic(err)
	} else {
		long, lat, r := planet.Position2000(julian.TimeToJD(dt))
		r *= AU
		return Spherical2Cartesian([]float64{r, long, lat}), []float64{math.Sqrt(2*c.μ/r - c.μ/c.a), long, lat}
	}

}

/* Definitions */

// Sun is our closest star.
var Sun = CelestialObject{"Sun", 695700, -1, 1.32712440018 * 1e20, -1, -1}

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 149598023, 3.986004415 * 1e5, 924645.0, 0.0010826269}

// Mars is the vacation place.
var Mars = CelestialObject{"Mars", 3397.2, 227939186, 4.305 * 1e4, 576000, 0.001964}
