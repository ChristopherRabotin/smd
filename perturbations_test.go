package smd

import (
	"testing"

	"github.com/gonum/floats"
)

func TestPertArbitrary(t *testing.T) {
	R := []float64{6524.834, 6862.875, 6448.296}
	V := []float64{4.901327, 5.533756, -1.976341}
	o := *NewOrbitFromRV(R, V, Earth)

	pertForce := []float64{1, 2, 3, 4, 5, 6, 0}

	arb := func(o Orbit, m Propagator) []float64 {
		return pertForce
	}

	perts := Perturbations{}
	perts.Arbitrary = arb

	if !floats.Equal(pertForce, perts.Perturb(o, GaussianVOP)) {
		t.Fatal("arbitrary pertubations fail")
	}

}
