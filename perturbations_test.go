package smd

import (
	"testing"
	"time"

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

	if !floats.Equal(pertForce, perts.Perturb(o, time.Now(), GaussianVOP)) {
		t.Fatal("arbitrary pertubations fail")
	}

}

func TestPert3rdBody(t *testing.T) {
	R := []float64{6524.834, 6862.875, 6448.296}
	V := []float64{4.901327, 5.533756, -1.976341}
	o := *NewOrbitFromRV(R, V, Earth)

	testValues := []struct {
		body CelestialObject
		pert []float64
	}{
		{Sun, []float64{-4.428955382575367e-10, 5.638380089266015e-10, 9.89098757275132e-11, 0, 0, 0, 0}},
		{Mars, []float64{-1.3044091305443318e-17, -9.991500206831446e-18, -1.0103860317125395e-17, 0, 0, 0, 0}},
		{Earth, []float64{0, 0, 0, 0, 0, 0, 0}},
	}

	perts := Perturbations{}
	dt, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	for _, test := range testValues {
		perts.PerturbingBody = &test.body
		pert := perts.Perturb(o, dt, Cartesian)
		if !floats.Equal(pert, test.pert) {
			t.Fatalf("invalid pertubations for %s\n%+v\n%v", test.body, pert, test.pert)
		}
	}

}
