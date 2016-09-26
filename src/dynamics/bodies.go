package dynamics

import (
	"fmt"

	"github.com/soniakeys/meeus/globe"
	"github.com/soniakeys/meeus/planetposition"
)

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	name  string
	mu    float64
	globe globe.Ellipsoid
	V87P  *planetposition.V87Planet
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.name)
}

/* Definitions */

var v87Earth, _ = planetposition.LoadPlanetPath(3, "../dataio/")

// Earth is home.
var Earth = CelestialObject{"Earth", 5.9742 * 1e24, globe.Earth76, v87Earth}
