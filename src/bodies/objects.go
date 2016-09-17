package bodies

import (
	"fmt"

	"github.com/soniakeys/meeus/elliptic"
	"github.com/soniakeys/meeus/globe"
)

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	name     string
	mass     int
	globe    *globe.Ellipsoid
	elements *elliptic.Elements
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.name)
}

/* Definitions */
// Earth is home.
var Earth = CelestialObject{"Earth", 5.97237e24, *globe.Earth76}
