package main

import (
	"fmt"
	"math"

	"dynamics"

	"github.com/ready-steady/ode/dopri"
)

func main() {
	fmt.Println("Testing 1D Dormant-Prince integrator.")
	dxdy1D := func(x float64, theta, f []float64) {
		fmt.Printf("theta=%+v\n", theta)
		f[0] = (-2.2067 * 1e-12) * (math.Pow(theta[0], 4) - 81*1e8)
	}
	//integrator, _ := dopri.New(&dopri.Config{TryStep: 4, MaxStep: 6, RelError: 1e-6, AbsError: 1e-12})
	integrator, _ := dopri.New(dopri.DefaultConfig())
	/*var xs = make([]float64, 16)
	xs[0] = 30
	for i := 1; i < 16; i++ {
		xs[i] = xs[i-1] + 30
	}*/
	xs := []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330, 360, 390, 420, 450, 480}
	y0s := []float64{1200}
	ys, _, err := integrator.Compute(dxdy1D, y0s, xs)
	if err != nil {
		panic(fmt.Errorf("1D dopri integration failed: %+v\n", err))
	}
	fmt.Printf("1D ys=%+v\n", ys)
	// TODO add verification statement.
	fmt.Println("Testing Dormant-Prince integrator on Euler EOMs.")
	// This will confirm that the angular moment magnitude is constant.
	maxAngMom := math.Inf(-1)
	minAngMom := math.Inf(1)
	att := dynamics.NewAttitude([3]float64{0.3, -0.4, 0.5}, [3]float64{0.1, 0.4, -0.2},
		[]float64{10, 0, 0, 0, 5, 0, 0, 0, 2})
	step := 0.01
	duration := 1
	numIterations := int(float64(duration) / step)
	var ts = make([]float64, numIterations)
	// Populate ts.
	for i := 1; i < numIterations; i++ {
		ts[i-1] = float64(i)
	}
	ys, xs2, err := integrator.Compute(att.EulerEOM(), att.State(), ts)
	if err != nil {
		panic(fmt.Errorf("Euler EOM dopri integration failed: %+v\n", err))
	}
	// Find the min and max angular moments.
	for _, y := range ys {
		maxAngMom = math.Max(maxAngMom, y)
		minAngMom = math.Min(minAngMom, y)
	}
	fmt.Printf("%+v\n\n---\n\n", ys)
	fmt.Printf("%+v\n\n---\n\n", xs2)
	fmt.Printf("Delta angular momentum = %+v\n", math.Abs(maxAngMom-minAngMom))
}
