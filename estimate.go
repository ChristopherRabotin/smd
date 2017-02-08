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
