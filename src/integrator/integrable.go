package integrator

// Integrable defines something which can be integrated, i.e. has a state vector.
// WARNING: Implementation must manage its own state based on the iteration.
type Integrable interface {
	GetState() []float64                   // Get the latest state of this integrable.
	SetState(i uint64, s []float64)        // Set the state s of a given iteration i.
	Stop(i uint64) bool                    // Return whether to stop the integration from iteration i.
	Func(t float64, s []float64) []float64 // ODE function from time t and state s, must return a new state.
}
