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

	arb := func(o Orbit) []float64 {
		return pertForce
	}

	perts := Perturbations{}
	perts.Arbitrary = arb

	if !floats.Equal(pertForce, perts.Perturb(o, time.Now(), Spacecraft{})) {
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
		{Sun, []float64{0, 0, 0, -4.4284739788758433e-10, 5.637851322253714e-10, 9.962451049697812e-11, 0}},
	}

	perts := Perturbations{}
	dt, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	for _, test := range testValues {
		perts.PerturbingBody = &test.body
		pert := perts.Perturb(o, dt, Spacecraft{})
		if !floats.EqualApprox(pert, test.pert, 1e-13) {
			t.Fatalf("invalid pertubations for %s\n%+v\n%v", test.body, pert, test.pert)
		}
	}

}
