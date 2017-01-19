package dynamics

import "testing"

func TestThrustControlI(t *testing.T) {
	_ = []ThrustControl{Inversion{}, Tangential{}, AntiTangential{}, OptimalThrust{}, new(OptimalΔOrbit)}
}
