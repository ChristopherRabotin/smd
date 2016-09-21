package integrator

import (
	"errors"
	"math/big"
)

// Integrable defines something which can be integrated, i.e. has a state vector.
// WARNING: Implementation must manage its own state based on the iteration.
type Integrable interface {
	GetState() []big.Float                        // Get the latest state of this integrable.
	SetState(i uint64, s []big.Float)             // Set the state s of a given iteration i.
	Stop(i uint64) bool                           // Return whether to stop the integration from iteration i.
	Func(t *big.Float, s []big.Float) []big.Float // ODE function from time t and state s, must return a new state.
}

// Config defines the configuration for the integration.
type Config struct {
	X0               big.Float  // The initial x0.
	StepSize         float64    // The step size.
	Integator        Integrable // What is to be integrated.
	stepSizeBigFloat *big.Float // The step size as a big float.
}

func (c *Config) verify() error {
	if c.StepSize <= 0 {
		return errors.New("config StepSize must be positive")
	}
	if c.Integator == nil {
		return errors.New("config Integator may not be nil")
	}
	c.stepSizeBigFloat = big.NewFloat(float64(c.StepSize))
	return nil
}

// SolveRK4 solves the provided ODE.
// Returns the number of iterations performed and the last X_i, or an error.
func SolveRK4(c *Config) (uint64, *big.Float, error) {
	iterNum := uint64(0)
	if err := c.verify(); err != nil {
		return iterNum, nil, err
	}

	xi := big.NewFloat(0.0).Copy(&c.X0)
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
			newState[i] = *big.NewFloat(0.0).Add(&state[i], big.NewFloat(0.0).Mul(oneSixth, &k1[i])).Add(big.NewFloat(0.0).Mul(oneThird, &k2[i]), big.NewFloat(0.0).Mul(oneThird, &k3[i])).Add(big.NewFloat(0.0), big.NewFloat(0.0).Mul(oneSixth, &k3[i]))
		}

		xi.Add(xi, c.stepSizeBigFloat)
		iterNum++ // Don't forget to increment the number of iterations.
	}

	return iterNum, nil, errors.New("this should not happen")
}
