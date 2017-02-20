package smd

import (
	"fmt"
	"math"
	"time"
)

// Perturbations defines how to handle perturbations during the propagation.
type Perturbations struct {
	Jn             uint8                                 // Factors to be used (only up to 4 supported)
	PerturbingBody *CelestialObject                      // The 3rd body which is perturbating the spacecraft.
	AutoThirdBody  bool                                  // Automatically determine what is the 3rd body based on distance and mass
	Arbitrary      func(o Orbit, m Propagator) []float64 // Additional arbitrary pertubation.
}

func (p Perturbations) isEmpty() bool {
	return p.Jn <= 1 && p.PerturbingBody == nil && p.AutoThirdBody && p.Arbitrary == nil
}

// Perturb returns the perturbing state vector based on the kind of propagation being used.
// For example, if using Cartesian, it'll return the impact on the R vector. If Gaussian, it'll
// return the impact on Ω, ω, ν (later for ν...).
func (p Perturbations) Perturb(o Orbit, dt time.Time, method Propagator) []float64 {
	pert := make([]float64, 7)
	if p.isEmpty() {
		return pert
	}
	if p.Jn > 1 && !o.Origin.Equals(Sun) {
		// Ignore any Jn about the Sun
		switch method {
		case GaussianVOP:
			a, _, _, _, _, _, _, _, _ := o.Elements()
			Ra := o.Origin.Radius / a
			acc := math.Sqrt(o.Origin.μ/math.Pow(o.Origin.Radius, 3)) * math.Pow(Ra, 7/2.)
			J2 := o.Origin.J(2)
			var J4 float64
			if p.Jn >= 4 {
				J4 = o.Origin.J(4)
			}
			// d\bar{Ω}/dt
			pert[3] += -acc * ((3/2.)*J2 - ((9/4.)*math.Pow(J2, 2)+(15/4.)*J4)*Ra)
			// d\bar{ω}/dt
			pert[4] += acc * ((3/2.)*J2 - (15/4.)*J4*Ra)
			// TODO: add effect on true anomaly.

		case Cartesian:
			R := o.R()
			r := norm(R)
			z2 := math.Pow(R[2], 2)
			acc := -(3 * o.Origin.μ * o.Origin.J(2) * math.Pow(o.Origin.Radius, 2)) / (2 * math.Pow(r, 5))
			pert[3] = acc * R[0] * (1 - 5*z2/(r*r))
			pert[4] = acc * R[1] * (1 - 5*z2/(r*r))
			pert[5] = acc * R[2] * (3 - 5*z2/(r*r))
			if p.Jn >= 3 {
				// Add J3 (computed via SageMath)
				x := R[0]
				y := R[1]
				z := R[2]
				r2 := math.Pow(R[0], 2) + math.Pow(R[1], 2) + z2
				r52 := math.Pow(r2, 5/2.)
				r72 := math.Pow(r2, 7/2.)
				r92 := math.Pow(r2, 9/2.)
				z3 := math.Pow(R[2], 3)
				z4 := math.Pow(R[2], 4)
				accJ3 := o.Origin.J(3) * math.Pow(o.Origin.Radius, 3) * o.Origin.μ
				pert[3] += (5 / 2.) * accJ3 * (7*x*z3/r92 - 3*x*z/r72)
				pert[4] += (5 / 2.) * accJ3 * (7*y*z3/r92 - 3*y*z/r72)
				pert[5] += 0.5 * accJ3 * (35*z4/r92 - 30*z2/r72 + 3/r52)
			}

		default:
			panic("unsupported propagation")
		}
	}
	if p.PerturbingBody != nil && !p.PerturbingBody.Equals(o.Origin) {
		switch method {
		case Cartesian:
			mainR := o.Origin.HelioOrbit(dt).R()
			pertR := p.PerturbingBody.HelioOrbit(dt).R()
			if p.PerturbingBody.Equals(Sun) {
				pertR = []float64{0, 0, 0}
			}
			relPertR := make([]float64, 3) // R between main body and pertubing body
			scPert := make([]float64, 3)   // r_{i/sc} of spacecraft to pertubing body.
			oppose := 1.
			if norm(mainR) > norm(pertR) {
				oppose = -1
			}
			scR := o.R()
			for i := 0; i < 3; i++ {
				relPertR[i] = oppose * (pertR[i] - mainR[i])
				scPert[i] = relPertR[i] - scR[i]
			}
			relPertRNorm3 := math.Pow(norm(relPertR), 3)
			scPertNorm3 := math.Pow(norm(scPert), 3)
			for i := 0; i < 3; i++ {
				pert[i] += p.PerturbingBody.μ * (scPert[i]/scPertNorm3 - relPertR[i]/relPertRNorm3)
			}

		default:
			panic(fmt.Errorf("third body perturbations not supported with %s propagator", method))
		}
	}
	if p.Arbitrary != nil {
		// Add the arbitrary perturbations
		arbs := p.Arbitrary(o, method)
		for i := 0; i < 7; i++ {
			pert[i] += arbs[i]
		}
	}
	return pert
}
