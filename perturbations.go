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
				// XXX: This is buggy!
				// TODO: Check my derivations
				/*
									--- FAIL: TestMissionGEOJ4 (0.07s)
					        mission_test.go:154:
					                oOsc: [-26499.555355218436 106200.89557088474 320939.8641177445]        [-1.7914458230504229 2.2873105390211204 8.373767217824176]
					                oTgt: [-42164.13611273549 -3.9738762194469346 0]        [0.00028978000140079556 -3.07466129972018 0]
					        mission_test.go:155:
					                oOsc: a=-5230.3 e=8.9668 i=71.420 Ω=23.732 ω=358.312 ν=88.564 λ=110.608 u=86.875
					                oTgt: a=42164.1 e=0.0000 i=0.000 Ω=359.993 ω=0.007 ν=180.005 λ=180.005 u=180.012
					        mission_test.go:156: [Cartesian] GEO 1.5 day propagation leads to incorrect orbit: true anomaly invalid
									FAIL
				*/
				// Add J3
				/*accJ3 := o.Origin.μ * math.Pow(o.Origin.Radius, 3) * o.Origin.J(3) / (2 * math.Pow(r, 6))
				pert[3] += accJ3 * 5 * R[0] * R[2] * (3 - 7*z2/(r*r))
				pert[4] += accJ3 * 5 * R[1] * R[2] * (3 - 7*z2/(r*r))
				pert[5] += accJ3 * (3*z2 - 3*math.Pow(r, 3)/5 + 3*z2/r - 7*math.Pow(R[2], 4)/math.Pow(r, 2))*/
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
