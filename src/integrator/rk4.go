package integrator

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
			tState[i] = state[i] + k1[i]*half
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k2[i] = y * r.StepSize
			tState[i] = state[i] + k2[i]*half
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k3[i] = y * r.StepSize
			tState[i] = state[i] + k3[i]
		}
		for i, y := range r.Integator.Func(xi+halfStep, tState) {
			k4[i] = y * r.StepSize
			newState[i] = state[i] + oneSixth*(k1[i]+k4[i]) + oneThird*(k2[i]+k3[i])
		}
		r.Integator.SetState(iterNum, newState)

		xi += r.StepSize
		iterNum++ // Don't forget to increment the number of iterations.
	}

	return iterNum, xi, nil
}