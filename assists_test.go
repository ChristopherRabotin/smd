package smd

import (
	"testing"

	"github.com/gonum/floats"
)

func TestBPlane(t *testing.T) {
	// The following values are correct according to Dr. Davis.
	rSOI := []float64{546507.344255845, -527978.380486028, 531109.066836708}
	vSOI := []float64{-4.9220589268733, 5.36316523097915, -5.22166308425181}
	orbit := NewOrbitFromRV(rSOI, vSOI, Earth)
	// Compute nominal values
	initBPlane := NewBPlane(*orbit)
	expBR := 10606.21042874
	expBT := 45892.32379544
	if !floats.EqualWithinAbs(initBPlane.BR, expBR, 1e-6) {
		t.Fatalf("BR got: %f\nexp:%f", initBPlane.BR, expBR)
	}
	if !floats.EqualWithinAbs(initBPlane.BT, expBT, 1e-6) {
		t.Fatalf("BT got: %f\nexp:%f", initBPlane.BT, expBT)
	}
}
