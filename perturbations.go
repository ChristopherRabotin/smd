package smd

import (
	"math"
	"math/rand"
	"time"

	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

// Perturbations defines how to handle perturbations during the propagation.
type Perturbations struct {
	Jn             uint8            // Factors to be used (only up to 4 supported)
	PerturbingBody *CelestialObject // The 3rd body which is perturbating the spacecraft.
	AutoThirdBody  bool             // Automatically determine what is the 3rd body based on distance and mass
	Drag           bool             // Set to true to use the Spacecraft's Drag for everything including STM computation
	Noise          OrbitNoise
	Arbitrary      func(o Orbit) []float64 // Additional arbitrary pertubation.
}

func (p Perturbations) isEmpty() bool {
	return p.Jn <= 1 && p.PerturbingBody == nil && p.AutoThirdBody && p.Arbitrary == nil
}

// STMSize returns the size of the STM
func (p Perturbations) STMSize() (r, c int) {
	if p.Drag {
		return 7, 7
	}
	return 6, 6
}

// Perturb returns the perturbing state vector based on the kind of propagation being used.
// For example, if using Cartesian, it'll return the impact on the R vector. If Gaussian, it'll
// return the impact on Ω, ω, ν (later for ν...).
func (p Perturbations) Perturb(o Orbit, dt time.Time, sc Spacecraft) []float64 {
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

	var RSunToEarth, RSunToSC, REarthToSC []float64

	if p.Drag || p.PerturbingBody != nil {
		REarthToSC = o.R()
		RSunToEarth = MxV33(R1(Deg2rad(-Earth.tilt)), o.Origin.HelioOrbit(dt).R())
		RSunToSC = make([]float64, 3)
		for i := 0; i < 3; i++ {
			RSunToSC[i] = RSunToEarth[i] + REarthToSC[i]
		}
	}

	if p.Drag {
		// If Drag, SRP is *also* turned on.
		// TODO: Drag, there is only SRP here.
		Cr := sc.Drag
		S := 0.01e-6 // TODO: Idem for the Area to mass ratio
		Phi := 1357.
		// Build the vectors.
		celerity := 2.997925e+05
		srpCst := (Phi * AU * AU * S / celerity) * Cr / math.Pow(Norm(RSunToSC), 3)
		for i := 0; i < 3; i++ {
			pert[i+3] += -srpCst * RSunToSC[i]
		}
	}

	if p.PerturbingBody != nil && !p.PerturbingBody.Equals(o.Origin) {
		if !p.PerturbingBody.Equals(Sun) {
			panic("only the Sun as a perturbing body is currently supported")
		}
		RSunToEarthNorm3 := math.Pow(Norm(RSunToEarth), 3)
		RSunToSCNorm3 := math.Pow(Norm(RSunToSC), 3)
		for i := 0; i < 3; i++ {
			pert[i+3] += Sun.μ * (RSunToEarth[i]/RSunToEarthNorm3 - RSunToSC[i]/RSunToSCNorm3)
		}
	}
	if p.Arbitrary != nil {
		// Add the arbitrary perturbations
		arbs := p.Arbitrary(o)
		for i := 0; i < 7; i++ {
			pert[i] += arbs[i]
		}
	}
	if p.Noise != (OrbitNoise{}) {
		orbitNoise := p.Noise.Generate()
		for i := 0; i < 6; i++ {
			pert[i] += orbitNoise[i]
		}
	}
	return pert
}

// OrbitNoise defines a new orbit noise applied as a perturbations.
// Use case is for generating datasets for filtering.
type OrbitNoise struct {
	probability float64
	position    *distmv.Normal
	velocity    *distmv.Normal
}

func (n OrbitNoise) Generate() (rtn []float64) {
	rtn = make([]float64, 6)
	if n.probability < rand.Float64() {
		return
	}
	position := n.position.Rand(nil)
	velocity := n.velocity.Rand(nil)
	for i := 0; i < 3; i++ {
		rtn[i] = position[i]
		rtn[i+3] = velocity[i]
	}
	return
}

func NewOrbitNoise(probability, sigmaPosition, sigmaVelocity float64) OrbitNoise {
	posMatrix := mat64.NewSymDense(3, []float64{sigmaPosition, 0, 0, 0, sigmaPosition, 0, 0, 0, sigmaPosition})
	velMatrix := mat64.NewSymDense(3, []float64{sigmaVelocity, 0, 0, 0, sigmaVelocity, 0, 0, 0, sigmaVelocity})
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	position, ok := distmv.NewNormal(make([]float64, 3), posMatrix, seed)
	if !ok {
		panic("process noise invalid")
	}
	velocity, ok := distmv.NewNormal(make([]float64, 3), velMatrix, seed)
	if !ok {
		panic("measurement noise invalid")
	}
	return OrbitNoise{probability, position, velocity}
}
