package dynamics

import (
	"math"

	"github.com/gonum/matrix/mat64"
)

// PQW2ECI converts a given vector from PQW frame to ECI frame.
func PQW2ECI(i, ω, Ω float64, vI []float64) (v []float64) {
	mulM := mat64.NewDense(3, 3, nil)
	mulM.Mul(R1(-i), R3(-ω))
	mulM.Mul(R3(-Ω), mulM)
	v = MxV33(mulM, vI)
	return
}

// R1 rotation about the 1st axis.
func R1(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{1, 0, 0, 0, c, s, 0, -s, c})
}

// R2 rotation about the 2nd axis.
func R2(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{c, 0, -s, 0, 1, 0, s, 0, c})
}

// R3 rotation about the 3rd axis.
func R3(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{c, s, 0, -s, c, 0, 0, 0, 1})
}
