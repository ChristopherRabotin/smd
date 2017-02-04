package smd

import (
	"math"
	"testing"

	"github.com/gonum/matrix/mat64"
)

func TestR1R2R3(t *testing.T) {
	x := math.Pi / 3.0
	s, c := math.Sincos(x)
	r1 := R1(x)
	r2 := R2(x)
	r3 := R3(x)
	// Test items equal to 1.
	if r1.At(0, 0) != r2.At(1, 1) || r1.At(0, 0) != r3.At(2, 2) || r3.At(2, 2) != 1 {
		t.Fatal("expected R1.At(0, 0) = R2.At(1, 1) = R3.At(2, 2) = 1\n")
	}
	// Test items equal to 0.
	if r1.At(0, 1) != r1.At(0, 2) || r1.At(1, 0) != r1.At(2, 0) || r1.At(0, 1) != 0 {
		t.Fatal("misplaced zeros in R1\n")
	}
	if r2.At(0, 1) != r2.At(1, 2) || r2.At(1, 0) != r2.At(1, 2) || r2.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R2\n")
	}
	if r3.At(2, 0) != r3.At(2, 1) || r3.At(0, 2) != r3.At(1, 2) || r3.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R3\n")
	}
	// Test R1.
	if r1.At(1, 1) != r1.At(2, 2) || r1.At(2, 2) != c {
		t.Fatal("expected R1 cosines misplaced\n")
	}
	if r1.At(2, 1) != -r1.At(1, 2) || r1.At(1, 2) != s {
		t.Fatal("expected R1 sines misplaced\n")
	}
	// Test R2.
	if r2.At(0, 0) != r2.At(2, 2) || r2.At(2, 2) != c {
		t.Fatal("expected R2 cosines misplaced\n")
	}
	if r2.At(2, 0) != -r2.At(0, 2) || r2.At(2, 0) != s {
		t.Fatal("expected R2 sines misplaced\n")
	}
	// Test R3.
	if r3.At(1, 1) != r3.At(0, 0) || r3.At(0, 0) != c {
		t.Fatal("expected R3 cosines misplaced\n")
	}
	if r3.At(0, 1) != -r3.At(1, 0) || r3.At(0, 1) != s {
		t.Fatal("expected R3 sines misplaced\n")
	}
}

func TestRot313(t *testing.T) {
	var R1R3, R3R1R3m mat64.Dense
	θ1 := math.Pi / 17
	θ2 := math.Pi / 16
	θ3 := math.Pi / 15
	R1R3.Mul(R1(θ2), R3(θ1))
	R3R1R3m.Mul(R3(θ3), &R1R3)
	R3R1R3m.Sub(&R3R1R3m, R3R1R3(θ1, θ2, θ3))
	if !mat64.Equal(&R3R1R3m, mat64.NewDense(3, 3, nil)) {
		t.Logf("\n%+v", mat64.Formatted(&R3R1R3m))
		t.Logf("\n%+v", mat64.Formatted(R3R1R3(θ1, θ2, θ3)))
		t.Fatal("failed")
	}
}

func TestPQW2ECI(t *testing.T) {
	i := Deg2rad(87.87)
	ω := Deg2rad(53.38)
	Ω := Deg2rad(227.89)
	Rp := Rot313Vec(i, ω, Ω, []float64{-466.7639, 11447.0219, 0})
	Re := []float64{6525.368103709379, 6861.531814548294, 6449.118636407358}
	if !vectorsEqual(Re, Rp) {
		t.Fatal("R conversion failed")
	}
	Vp := Rot313Vec(i, ω, Ω, []float64{-5.996222, 4.753601, 0})
	Ve := []float64{4.902278620687254, 5.533139558121602, -1.9757104281719946}
	if !vectorsEqual(Ve, Vp) {
		t.Fatal("V conversion failed")
	}
}
