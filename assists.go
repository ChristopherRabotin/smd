package smd

import "math"

// GATurnAngle computes the turn angle about a given body based on the radius of periapsis.
func GATurnAngle(vInf, rP float64, body CelestialObject) float64 {
	ρ := math.Acos(1 / (1 + math.Pow(vInf, 2)*(rP/body.μ)))
	return math.Pi - 2*ρ
}
