package smd

import "math"

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
func (p Perturbations) Perturb(o Orbit, method Propagator) []float64 {
	pert := make([]float64, 7)
	if p.isEmpty() {
		return pert
	}
	if p.Jn > 1 && !o.Origin.Equals(Sun) {
		// Ignore any Jn about the Sun
		switch method {
		case GaussianVOP:
			Ra := o.Origin.Radius / o.a
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
			acc := -(3 * o.Origin.μ * o.Origin.J2 * math.Pow(o.Origin.Radius, 2)) / (2 * math.Pow(r, 5))
			pert[3] = acc * R[0] * (1 - 5*z2/(r*r))
			pert[4] = acc * R[1] * (1 - 5*z2/(r*r))
			pert[5] = acc * R[2] * (3 - 5*z2/(r*r))

		default:
			panic("unsupported propagation")
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
