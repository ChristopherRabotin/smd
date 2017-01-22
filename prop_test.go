package smd

import (
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestThrustControlI(t *testing.T) {
	_ = []ThrustControl{Inversion{}, Tangential{}, AntiTangential{}, OptimalThrust{}, new(OptimalΔOrbit)}
}

func TestHohmannΔv(t *testing.T) {
	target := NewOrbitFromOE(Earth.Radius+35781.34857, 0, 0, 0, 0, 90, Earth)
	init := NewOrbitFromOE(Earth.Radius+191.34411, 0, 0, 0, 0, 90, Earth)
	transfer := NewHohmannΔv(*target)
	transfer.Precompute(*init)
	ΔvApoExp := []float64{0.0, -1.478187, 0.0}
	ΔvPeriExp := []float64{0.0, 2.457038, 0.0}
	tofExp := time.Duration(5)*time.Hour + time.Duration(15)*time.Minute + time.Duration(24)*time.Second
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(ΔvApoExp[i], transfer.ΔvApo[i], velocityε) {
			t.Fatalf("ΔvApo[%d] failed: %f != %f", i, ΔvApoExp[i], transfer.ΔvApo[i])
		}
		if !floats.EqualWithinAbs(ΔvPeriExp[i], transfer.ΔvPeri[i], velocityε) {
			t.Fatalf("ΔvPeri[%d] failed: %f != %f", i, ΔvPeriExp[i], transfer.ΔvPeri[i])
		}
	}
	if transfer.tof != tofExp {
		t.Fatalf("invalid TOF: %d != %d", transfer.tof, tofExp)
	}
}
