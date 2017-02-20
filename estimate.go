package smd

import (
	"math"
	"os"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	kitlog "github.com/go-kit/kit/log"
	"github.com/gonum/matrix/mat64"
)

// OrbitEstimate is an ode.Integrable which allows to propagate an orbit via its initial estimate.
type OrbitEstimate struct {
	Φ      *mat64.Dense  // STM
	Orbit  Orbit         // estimated orbit
	Perts  Perturbations // perturbations to account for
	StopDT time.Time     // end time of te integration
	dt     time.Time     // current time of the integration
	step   time.Duration // time step
	logger kitlog.Logger // logger
}

// GetState gets the state.
func (e *OrbitEstimate) GetState() []float64 {
	s := make([]float64, 6)
	R, V := e.Orbit.RV()
	s[0] = R[0]
	s[1] = R[1]
	s[2] = R[2]
	s[3] = V[0]
	s[4] = V[1]
	s[5] = V[2]
	// Add the components of Φ
	sIdx := 6
	rΦ, cΦ := e.Φ.Dims()
	for i := 0; i < rΦ; i++ {
		for j := 0; j < cΦ; j++ {
			s[sIdx] = e.Φ.At(i, j)
			sIdx++
		}
	}
	return s
}

// SetState sets the next state at time t.
func (e *OrbitEstimate) SetState(t float64, s []float64) {
	R := []float64{s[0], s[1], s[2]}
	V := []float64{s[3], s[4], s[5]}
	e.Orbit = *NewOrbitFromRV(R, V, e.Orbit.Origin)
	// Extract the components of Φ
	sIdx := 6
	rΦ, cΦ := e.Φ.Dims()
	for i := 0; i < rΦ; i++ {
		for j := 0; j < cΦ; j++ {
			e.Φ.Set(i, j, s[sIdx])
			sIdx++
		}
	}
	// Increment the time.
	e.dt = e.dt.Add(e.step)
}

// Stop returns whether we should stop the integration.
func (e *OrbitEstimate) Stop(t float64) bool {
	return true
}

// Func does the math. Returns a new state.
func (e *OrbitEstimate) Func(t float64, f []float64) (fDot []float64) {
	// XXX: Note that this function is very similar to Mission.Func for a Cartesian propagation.
	// *BUT* we need to add in all the components of Φ, since they have to be integrated too.
	fDot = make([]float64, 6) // init return vector
	// Re-create the orbit from the state.
	R := []float64{f[0], f[1], f[2]}
	V := []float64{f[3], f[4], f[5]}
	orbit := NewOrbitFromRV(R, V, e.Orbit.Origin)
	bodyAcc := -orbit.Origin.μ / math.Pow(orbit.RNorm(), 3)
	// d\vec{R}/dt
	fDot[0] = f[3]
	fDot[1] = f[4]
	fDot[2] = f[5]
	// d\vec{V}/dt
	fDot[3] = bodyAcc * f[0]
	fDot[4] = bodyAcc * f[1]
	fDot[5] = bodyAcc * f[2]

	pert := e.Perts.Perturb(*orbit, e.dt, Cartesian)
	for i := 0; i < 7; i++ {
		fDot[i] += pert[i]
	}

	// Extract the components of Φ
	fIdx := 6
	rΦ, cΦ := e.Φ.Dims()
	Φ := mat64.NewDense(rΦ, cΦ, nil)
	ΦDot := mat64.NewDense(rΦ, cΦ, nil)
	for i := 0; i < rΦ; i++ {
		for j := 0; j < cΦ; j++ {
			Φ.Set(i, j, f[fIdx])
			fIdx++
		}
	}

	// Compute the STM.
	A := mat64.NewDense(6, 6, nil)
	A.Set(0, 3, 1)
	A.Set(1, 4, 1)
	A.Set(2, 5, 1)
	// Jn perturbations:
	if e.Perts.Jn > 1 {
		// Ai0 = \frac{\partial a}{\partial x}
		// Ai1 = \frac{\partial a}{\partial y}
		// Ai2 = \frac{\partial a}{\partial z}
		A30 := A.At(3, 0)
		A40 := A.At(4, 0)
		A50 := A.At(5, 0)
		A31 := A.At(3, 1)
		A41 := A.At(4, 1)
		A51 := A.At(5, 1)
		A32 := A.At(3, 2)
		A42 := A.At(4, 2)
		A52 := A.At(5, 2)

		// Notation simplification
		x := R[0]
		y := R[1]
		z := R[2]
		x2 := math.Pow(R[0], 2)
		y2 := math.Pow(R[1], 2)
		z2 := math.Pow(R[2], 2)
		z3 := math.Pow(R[2], 3)
		z4 := math.Pow(R[2], 4)
		r2 := x2 + y2 + z2
		// Adding those fractions to avoid forgetting the trailing period which makes them floats.
		f32 := 3 / 2.
		f152 := 15 / 2.
		r252 := math.Pow(r2, 5/2.)
		r272 := math.Pow(r2, 7/2.)
		r292 := math.Pow(r2, 9/2.)
		// J2
		j2fact := e.Orbit.Origin.J(2) * math.Pow(e.Orbit.Origin.Radius, 2) * e.Orbit.Origin.μ
		A30 += -f32 * j2fact * (35*x2*z2/r292 - 5*x2/r272 - 5*z2/r272 + 1/r252) //dAxDx
		A40 += -f152 * j2fact * (7*x*y*z2/r292 - x*y/r272)                      //dAyDx
		A50 += -f152 * j2fact * (7*x*z3/r292 - 3*x*z/r272)                      //dAzDx

		A31 += -f152 * j2fact * (7*x*y*z2/r292 - x*y/r272)                      //dAxDy
		A41 += -f32 * j2fact * (35*y2*z2/r292 - 5*y2/r272 - 5*z2/r272 + 1/r252) // dAyDy
		A51 += -f152 * j2fact * (7*y*z3/r292 - 3*y*z/r272)                      // dAzDy

		A32 += -f152 * j2fact * (7*x*z3/r292 - 3*x*z/r272)        //dAxDz
		A42 += -f152 * j2fact * (7*y*z3/r292 - 3*y*z/r272)        //dAyDz
		A52 += -f32 * j2fact * (35*z4/r292 - 30*z2/r272 + 3/r252) // dAzDz

		// J3
		if e.Perts.Jn > 2 {
			z5 := math.Pow(R[2], 5)
			r2112 := math.Pow(r2, 11/2.)
			f52 := 5 / 2.
			f1052 := 105 / 2.
			j3fact := e.Orbit.Origin.J(3) * math.Pow(e.Orbit.Origin.Radius, 3) * e.Orbit.Origin.μ
			A30 += -f52 * j3fact * (63*x2*z3/r2112 - 21*x2*z/r292 - 7*z3/r292 + 3*z/r272) //dAxDx
			A40 += -f1052 * j3fact * (3*x*y*z3/r2112 - x*y*z/r292)                        //dAyDx
			A50 += -f152 * j3fact * (21*x*z4/r2112 - 14*x*z2/r292 + x/r272)               //dAzDx

			A31 += -f1052 * j3fact * (3*x*y*z3/r2112 - x*y*z/r292)                        //dAxDy
			A41 += -f52 * j3fact * (63*y2*z3/r2112 - 21*y2*z/r292 - 7*z3/r292 + 3*z/r272) // dAyDy
			A51 += -f152 * j3fact * (21*y*z4/r2112 - 14*y*z2/r292 + y/r272)               // dAzDy

			A32 += -f152 * j3fact * (21*x*z4/r2112 - 14*x*z2/r292 + x/r272) //dAxDz
			A42 += -f152 * j3fact * (21*y*z4/r2112 - 14*y*z2/r292 + y/r272) //dAyDz
			A52 += -f52 * j3fact * (63*z5/r2112 - 70*z3/r292 + 15*z/r272)   // dAzDz
		}
		// \frac{\partial a}{\partial x}
		A.Set(3, 0, A30)
		A.Set(4, 0, A40)
		A.Set(5, 0, A50)
		// \partial a/\partial y
		A.Set(3, 1, A31)
		A.Set(4, 1, A41)
		A.Set(5, 1, A51)
		// \partial a/\partial z
		A.Set(3, 2, A32)
		A.Set(4, 2, A42)
		A.Set(5, 2, A52)

	}

	ΦDot.Mul(A, Φ)

	// Store ΦDot in fDot
	fIdx = 6
	for i := 0; i < rΦ; i++ {
		for j := 0; j < cΦ; j++ {
			fDot[fIdx] = ΦDot.At(i, j)
			fIdx++
		}
	}
	return fDot
}

// NewOrbitEstimate returns a new Estimate of an orbit given the perturbations to be taken into account.
// The only supported state is [\vec{r} \vec{v}]^T (for now at least).
func NewOrbitEstimate(n string, o Orbit, p Perturbations, epoch time.Time, duration, step time.Duration) *OrbitEstimate {
	// The initial previous STM is identity.
	klog := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	klog = kitlog.NewContext(klog).With("estimate", n)
	stopDT := epoch.Add(duration)
	est := OrbitEstimate{gokalman.DenseIdentity(6), o, p, stopDT, epoch, step, klog}

	if p.Jn > 2 {
		est.logger.Log("severity", "warning", "msg", "only J2 supported")
	}
	return &est
}
