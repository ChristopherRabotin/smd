package smd

import "testing"

func TestThrustControlI(t *testing.T) {
	_ = []ThrustControl{Tangential{}, AntiTangential{}, OptimalThrust{}, new(OptimalΔOrbit)}
}
