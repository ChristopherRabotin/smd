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
		jfact := e.Orbit.Origin.μ * math.Pow(e.Orbit.Origin.Radius, 2) / 2
		// Notation simplification
		x := R[0]
		y := R[1]
		z := R[2]
		r := orbit.RNorm()
		J2 := e.Orbit.Origin.J2
		j2fact1 := -15 * J2 / math.Pow(r, 6)
		j2fact2 := 3 * J2 / math.Pow(r, 5)
		f5zr := 5 * math.Pow(z, 2) / math.Pow(r, 2)
		f10xz := -10 * x * math.Pow(z, 2) / math.Pow(r, 3)
		f10yz := 10 * y * math.Pow(z, 2) / math.Pow(r, 3) // Yes, this is +10 ...
		rzr := (r - math.Pow(z, 2)) / math.Pow(r, 3)
		// Ai0 = \frac{\partial a}{\partial x}
		// Ai1 = \frac{\partial a}{\partial y}
		// Ai2 = \frac{\partial a}{\partial z}
		// J2
		A30 := A.At(3, 0)
		A30 += j2fact1*(1-f5zr)*math.Pow(x, 2) + j2fact2*(1-f5zr+f10xz*x)
		A40 := A.At(4, 0)
		A40 += j2fact1*(1-f5zr)*x*y + j2fact2*(f10xz*y)
		A50 := A.At(5, 0)
		A50 += j2fact1*(3-f5zr)*x*z + j2fact2*(f10xz*z)

		A31 := A.At(3, 1)
		A31 += j2fact1*(1-f5zr)*x*y + j2fact2*(f10yz*x)
		A41 := A.At(4, 1)
		A41 += j2fact1*(1-f5zr)*x*y*y + j2fact2*(1-f5zr+f10yz*y)
		A51 := A.At(5, 1)
		A51 += j2fact1*(3-f5zr)*x*z*y + j2fact2*(f10yz*z)

		A32 := A.At(3, 2)
		A32 += j2fact1*(1-f5zr)*x*z + j2fact2*10*x*z*rzr
		A42 := A.At(4, 2)
		A42 += j2fact1*(1-f5zr)*y*z + j2fact2*10*y*z*rzr
		A52 := A.At(5, 2)
		A52 += j2fact1*(3-f5zr)*math.Pow(z, 2) + j2fact2*(10*math.Pow(z, 2)*rzr+3-f5zr)
		// J3
		if e.Perts.Jn > 2 {
			J3 := e.Orbit.Origin.J3
			j3fact1 := -6 * J3 / math.Pow(r, 7)
			j3fact2 := J3 / math.Pow(r, 6)
			f7zr := 7 * math.Pow(z, 2) / math.Pow(r, 2)

			// TODO: Use results from SageMath sheet
			A30 += j3fact1*x*(5*x*z*(3-f7zr)) + j3fact2*(70*math.Pow(x, 2)*math.Pow(z, 3)/math.Pow(r, 3)+5*z*(3-f7zr))
			A40 += j3fact1*x*(5*y*z*(3-f7zr)) + j3fact2*(70*x*y*math.Pow(z, 3)/math.Pow(r, 3)+5*z*(3-f7zr))
			A50 += j3fact1*x*(3*math.Pow(z, 2)*(1+1/r)-3*math.Pow(r, 3)/5-f7zr*math.Pow(z, 2)) + j3fact2*x*(2*math.Pow(z, 2)*f7zr-6*math.Pow(z, 2)/math.Pow(r, 2)-9*math.Pow(r, 2)/5)

			A31 += j3fact1*y*(5*x*z*(3-f7zr)) + j3fact2*(5*x*z*f7zr*2*y)
			A41 += j3fact1*y*(5*y*z*(3-f7zr)) + j3fact2*(70*math.Pow(y, 2)*math.Pow(z, 3)/math.Pow(r, 3)+5*z*(3-f7zr))
			A51 += j3fact1*y*(3*math.Pow(z, 2)*(1+1/r)-3*math.Pow(r, 3)/5-f7zr*math.Pow(z, 2)) + j3fact2*y*(2*math.Pow(z, 2)*f7zr-6*math.Pow(z, 2)/math.Pow(r, 2)-9*math.Pow(r, 2)/5)

			A32 += j3fact1*z*(5*x*z*(3-f7zr)) + j3fact2*(5*x*(3-f7zr)+70*x*math.Pow(z, 2)*rzr)
			A42 += j3fact1*z*(5*y*z*(3-f7zr)) + j3fact2*(5*y*(3-f7zr)+70*y*math.Pow(z, 2)*rzr)
			A52 += j3fact1*z*(3*math.Pow(z, 2)*(1+1/r)-3*math.Pow(r, 3)/5-f7zr*math.Pow(z, 2)) + j3fact2*(6*z+14*z*rzr+(6*z*r-3*math.Pow(z, 3))/math.Pow(r, 2))
		}
		// \frac{\partial a}{\partial x}
		A.Set(3, 0, jfact*A30)
		A.Set(4, 0, jfact*A40)
		A.Set(5, 0, jfact*A50)
		// \partial a/\partial y
		A.Set(3, 1, jfact*A31)
		A.Set(4, 1, jfact*A41)
		A.Set(5, 1, jfact*A51)
		// \partial a/\partial z
		A.Set(3, 2, jfact*A32)
		A.Set(4, 2, jfact*A42)
		A.Set(5, 2, jfact*A52)

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
