package dynamics

import "fmt"

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	Name   string
	Radius float64
	μ      float64
	SOI    float64 // With respect to the Sun
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.Name)
}

/* Definitions */

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 3.986004415 * 1e5, 924645.0}
