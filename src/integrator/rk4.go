package integrator

import (
	"fmt"
	"math/big"
)

// Integrable defines something which can be integrated, i.e. has a state vector.
// WARNING: Implementation must manage its own state based on the iteration.
type Integrable interface {
	GetState() []float64                   // Get the latest state of this integrable.
	SetState(i uint64, s []float64)        // Set the state s of a given iteration i.
	Stop(i uint64) bool                    // Return whether to stop the integration from iteration i.
	Func(t float64, s []float64) []float64 // ODE function from time t and state s, must return a new state.
}

// RK4 defines an RK4 integrator using math.Big Floats.
type RK4 struct {
	X0        float64    // The initial x0.
	StepSize  float64    // The step size.
	Integator Integrable // What is to be integrated.
}

// NewRK4 returns a new BigRK4 integrator instance.
func NewRK4(x0 float64, stepSize float64, inte Integrable) (r *RK4) {
	if stepSize <= 0 {
		panic("config StepSize must be positive")
	}
	if inte == nil {
		panic("config Integator may not be nil")
	}
	r = &RK4{X0: x0, StepSize: stepSize, Integator: inte}
	return
}

// Solve solves the configured RK4.
// Returns the number of iterations performed and the last X_i, or an error.
func (r *RK4) Solve() (uint64, float64, error) {
	const (
		half     = 0.5
		oneSixth = 1 / 6.0
		oneThird = 1 / 3.0
	)

	iterNum := uint64(0)
	xi := r.X0
	for !r.Integator.Stop(iterNum) {
		halfStep := xi * half
		state := r.Integator.GetState()
		newState := make([]float64, len(state))
		k1 := make([]float64, len(state))
		//k2, k3, k4 are used as buffers AND result variables.
		k2 := make([]float64, len(state))
		k3 := make([]float64, len(state))
		k4 := make([]float64, len(state))
		tState := make([]float64, len(state))

		// Compute the k's.
		for i, y := range r.Integator.Func(xi, state) {
			k1[i] = y * r.StepSize
			fmt.Printf("k1 = %4.4f\n", k1[i])
			tState[i] = state[i] + k1[i]*half
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k2[i] = y * r.StepSize
			fmt.Printf("k2 = %4.4f\n", k2[i])
			tState[i] = state[i] + k2[i]*half
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k3[i] = y * r.StepSize
			fmt.Printf("k3 = %4.4f\n", k3[i])
			tState[i] = state[i] + k3[i]
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k4[i] = y * r.StepSize
			fmt.Printf("k4 = %4.4f\n", k4[i])
		}
		// Let's now compute the new state.
		for i := range newState {
			newState[i] = state[i] + oneSixth*(k1[i]+k4[i]) + oneThird*(k2[i]+k3[i])
		}
		r.Integator.SetState(iterNum, newState)

		xi += r.StepSize
		iterNum++ // Don't forget to increment the number of iterations.
	}

	return iterNum, xi, nil
}

/* --- *
* Integration using math.big.
* ---- */

// BigIntegrable defines something which can be integrated, i.e. has a state vector.
// WARNING: Implementation must manage its own state based on the iteration.
type BigIntegrable interface {
	GetState() []big.Float                        // Get the latest state of this integrable.
	SetState(i uint64, s []big.Float)             // Set the state s of a given iteration i.
	Stop(i uint64) bool                           // Return whether to stop the integration from iteration i.
	Func(t *big.Float, s []big.Float) []big.Float // ODE function from time t and state s, must return a new state.
}

// BigRK4 defines an RK4 integrator using math.Big Floats.
type BigRK4 struct {
	X0               *big.Float    // The initial x0.
	StepSize         float64       // The step size.
	Integator        BigIntegrable // What is to be integrated.
	stepSizeBigFloat *big.Float    // The step size as a big float.
}

// NewBigRK4 returns a new BigRK4 integrator instance.
func NewBigRK4(x0 *big.Float, stepSize float64, inte BigIntegrable) (r *BigRK4) {

	if stepSize <= 0 {
		panic("config StepSize must be positive")
	}
	if inte == nil {
		panic("config Integator may not be nil")
	}
	r = &BigRK4{X0: x0, StepSize: stepSize, Integator: inte}
	r.stepSizeBigFloat = big.NewFloat(float64(stepSize))
	return
}

// Solve solves the configured RK4.
// Returns the number of iterations performed and the last X_i, or an error.
func (c *BigRK4) Solve() (uint64, *big.Float, error) {
	iterNum := uint64(0)

	xi := big.NewFloat(0.0).Copy(c.X0)
	half := big.NewFloat(.5)
	oneSixth := big.NewFloat(1 / 6)
	oneThird := big.NewFloat(1 / 3)
	halfStep := big.NewFloat(0.0).Mul(xi, half)

	for !c.Integator.Stop(iterNum) {
		state := c.Integator.GetState()
		newState := make([]big.Float, len(state))
		k1 := make([]big.Float, len(state))
		//k2, k3, k4 are used as buffers AND result variables.
		k2 := make([]big.Float, len(state))
		k3 := make([]big.Float, len(state))
		k4 := make([]big.Float, len(state))
		// Compute the k's.
		for i, k1n := range c.Integator.Func(xi, state) {
			k1[i].Mul(&k1n, c.stepSizeBigFloat)
			k2[i].Mul(&k1[i], half)
			k2[i].Add(&k2[i], c.stepSizeBigFloat)
			k1af, _ := k1[i].Float64()
			fmt.Printf("k1 = %4.4f\n", k1af)
		}
		for i, k2n := range c.Integator.Func(xi.Add(xi, halfStep), k2) {
			k2[i].Mul(&k2n, c.stepSizeBigFloat)
			k3[i].Mul(&k2[i], half)
			k3[i].Add(&k3[i], c.stepSizeBigFloat)
		}
		for i, k3n := range c.Integator.Func(xi.Add(xi, halfStep), k3) {
			k3[i].Mul(&k3n, c.stepSizeBigFloat)
			k4[i].Mul(&k2[i], half)
			k4[i].Add(&k3[i], c.stepSizeBigFloat)
		}
		for i, k4n := range c.Integator.Func(xi.Add(xi, halfStep), k4) {
			k4[i].Mul(&k4n, c.stepSizeBigFloat)
		}
		// Let's now compute the new state.
		for i := range newState {
			newState[i] = *big.NewFloat(0.0).Add(&state[i], big.NewFloat(1.0).Mul(oneSixth, &k1[i])).Add(big.NewFloat(1.0).Mul(oneThird, &k2[i]), big.NewFloat(1.0).Mul(oneThird, &k3[i])).Add(big.NewFloat(0.0), big.NewFloat(1.0).Mul(oneSixth, &k3[i]))
		}
		c.Integator.SetState(iterNum, newState)

		xi.Add(xi, c.stepSizeBigFloat)
		iterNum++ // Don't forget to increment the number of iterations.
	}

	return iterNum, xi, nil
}
