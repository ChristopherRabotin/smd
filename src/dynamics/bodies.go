package dynamics

import (
	"fmt"
	"math/big"

	"github.com/soniakeys/meeus/elliptic"
	"github.com/soniakeys/meeus/globe"
	"github.com/soniakeys/meeus/planetposition"
)

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	name     string
	mass     *big.Float
	globe    globe.Ellipsoid
	elements *elliptic.Elements
	V87P     *planetposition.V87Planet
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.name)
}

/* Definitions */

var v87Earth, _ = planetposition.LoadPlanetPath(3, "../dataio/")

// Earth is home.
var Earth = CelestialObject{"Earth", big.NewFloat(5.97237e24), globe.Earth76, nil, v87Earth}
