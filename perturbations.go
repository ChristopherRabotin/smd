package smd

import (
	"math"
	"time"
)

// Perturbations defines how to handle perturbations during the propagation.
type Perturbations struct {
	Jn             uint8                   // Factors to be used (only up to 4 supported)
	PerturbingBody *CelestialObject        // The 3rd body which is perturbating the spacecraft.
	AutoThirdBody  bool                    // Automatically determine what is the 3rd body based on distance and mass
	Arbitrary      func(o Orbit) []float64 // Additional arbitrary pertubation.
}

func (p Perturbations) isEmpty() bool {
	return p.Jn <= 1 && p.PerturbingBody == nil && p.AutoThirdBody && p.Arbitrary == nil
}

// Perturb returns the perturbing state vector based on the kind of propagation being used.
// For example, if using Cartesian, it'll return the impact on the R vector. If Gaussian, it'll
// return the impact on Ω, ω, ν (later for ν...).
func (p Perturbations) Perturb(o Orbit, dt time.Time) []float64 {
	pert := make([]float64, 7)
	if p.isEmpty() {
		return pert
	}
	if p.Jn > 1 && !o.Origin.Equals(Sun) {
		// Ignore any Jn about the Sun
		R := o.R()
		x := R[0]
		y := R[1]
		z := R[2]
		z2 := math.Pow(R[2], 2)
		z3 := math.Pow(R[2], 3)
		r2 := math.Pow(R[0], 2) + math.Pow(R[1], 2) + z2
		r252 := math.Pow(r2, 5/2.)
		r272 := math.Pow(r2, 7/2.)
		// J2 (computed via SageMath: https://cloud.sagemath.com/projects/1fb6b227-1832-4f82-a05c-7e45614c00a2/files/j2perts.sagews)
		accJ2 := (3 / 2.) * o.Origin.J(2) * math.Pow(o.Origin.Radius, 2) * o.Origin.μ
		pert[3] += accJ2 * (5*x*z2/r272 - x/r252)
		pert[4] += accJ2 * (5*y*z2/r272 - y/r252)
		pert[5] += accJ2 * (5*z3/r272 - 3*z/r252)
		if p.Jn >= 3 {
			// J3 (computed via SageMath: https://cloud.sagemath.com/#projects/1fb6b227-1832-4f82-a05c-7e45614c00a2/files/j3perts.sagews)
			r292 := math.Pow(r2, 9/2.)
			z4 := math.Pow(R[2], 4)
			accJ3 := o.Origin.J(3) * math.Pow(o.Origin.Radius, 3) * o.Origin.μ
			pert[3] += (5 / 2.) * accJ3 * (7*x*z3/r292 - 3*x*z/r272)
			pert[4] += (5 / 2.) * accJ3 * (7*y*z3/r292 - 3*y*z/r272)
			pert[5] += 0.5 * accJ3 * (35*z4/r292 - 30*z2/r272 + 3/r252)
		}
	}
	if p.PerturbingBody != nil && !p.PerturbingBody.Equals(o.Origin) {

		mainR := o.Origin.HelioOrbit(dt).R()
		pertR := p.PerturbingBody.HelioOrbit(dt).R()
		if p.PerturbingBody.Equals(Sun) {
			pertR = []float64{0, 0, 0}
		}
		relPertR := make([]float64, 3) // R between main body and pertubing body
		scPert := make([]float64, 3)   // r_{i/sc} of spacecraft to pertubing body.
		oppose := 1.
		if Norm(mainR) > Norm(pertR) {
			oppose = -1
		}
		scR := o.R()
		for i := 0; i < 3; i++ {
			relPertR[i] = oppose * (pertR[i] - mainR[i])
			scPert[i] = relPertR[i] - scR[i]
		}
		relPertRNorm3 := math.Pow(Norm(relPertR), 3)
		scPertNorm3 := math.Pow(Norm(scPert), 3)
		for i := 0; i < 3; i++ {
			pert[i] += p.PerturbingBody.μ * (scPert[i]/scPertNorm3 - relPertR[i]/relPertRNorm3)
		}
	}
	if p.Arbitrary != nil {
		// Add the arbitrary perturbations
		arbs := p.Arbitrary(o)
		for i := 0; i < 7; i++ {
			pert[i] += arbs[i]
		}
	}
	return pert
}
