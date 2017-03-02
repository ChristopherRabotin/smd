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
	// Let's test the B-Plane correction too.
	initBPlane.SetBRGoal(5022.26511510685, 1e-6)
	initBPlane.SetBTGoal(13135.7982982557, 1e-6)
	finalV, err := initBPlane.AchieveGoals(2)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	expV := []float64{-5.222055735935133, 5.221567577651425, -5.22166308425181}
	for i := 0; i < 3; i++ {
		if !floats.EqualWithinAbs(expV[i], finalV[i], 1e-8) {
			t.Fatal("invalid TCM computed")
		}
	}
}
