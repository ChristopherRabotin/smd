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
			// TODO: Switch to the more complete description.
			sp := o.SemiParameter()
			cosi := math.Cos(o.i)
			// d\bar{Ω}/dt
			pert[3] += -(3 * math.Sqrt(o.Origin.μ/math.Pow(o.a, 3)) * o.Origin.J2 / 2) * math.Pow(o.Origin.Radius/sp, 2) * cosi
			// d\bar{ω}/dt
			pert[4] += -(3 * math.Sqrt(o.Origin.μ/math.Pow(o.a, 3)) * o.Origin.J2 / 4) * math.Pow(o.Origin.Radius/sp, 2) * (5*math.Pow(cosi, 2) - 1)
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
