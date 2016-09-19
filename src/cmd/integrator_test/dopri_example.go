package main

import (
	"fmt"
	"log"
	"math"

	"dynamics"

	"github.com/ready-steady/ode/dopri"
)

func main() {
	fmt.Println("Testing 1D Dormant-Prince integrator.")
	dxdy1D := func(x float64, theta, f []float64) {
		f[0] = (-2.2067 * 1e-12) * (math.Pow(theta[0], 4) - 81*1e8)
	}
	integrator, _ := dopri.New(dopri.DefaultConfig())
	var xs = make([]float64, 16)
	xs[0] = 30
	for i := 1; i < 16; i++ {
		xs[i] = xs[i-1] + 30
	}
	_, _, err := integrator.Compute(dxdy1D, []float64{1200}, xs)
	if err != nil {
		log.Fatalf("integration failed: %+v\n", err)
	}
	// TODO add verification statement.
	fmt.Println("Testing Dormant-Prince integrator on Euler EOMs.")
	// This will confirm that the angular moment magnitude is constant.
	maxAngMom := math.Inf(-1)
	minAngMom := math.Inf(1)
	// stateVector is defined as sigma and omega.
	stateVector := []float64{0.3, -0.4, 0.5, 0.1, 0.4, -0.2}
	dynamics.NewAttitude()

}
