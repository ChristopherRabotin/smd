package dynamics

import (
	"math"

	"github.com/gonum/matrix/mat64"
)

// PQW2ECI converts a given vector from PQW frame to ECI frame.
func PQW2ECI(i, ω, Ω float64, vI []float64) []float64 {
	//return MxV33(R3R1R3(-i, -ω, -Ω), vI) // TODO: Do the math manually, Vallado is probably wrong.
	var mulM mat64.Dense
	mulM.Mul(R3(-ω), R1(-i))
	mulM.Mul(&mulM, R3(-Ω))
	return MxV33(&mulM, vI)
}

// R3R1R3 simplifies PQW2ECI.
func R3R1R3(i, ω, Ω float64) *mat64.Dense {
	si, ci := math.Sincos(i)
	sω, cω := math.Sincos(ω)
	sΩ, cΩ := math.Sincos(Ω)
	return mat64.NewDense(3, 3, []float64{cΩ*cω - sΩ*sω*ci, -1*cΩ*sω - sΩ*cω*ci, sΩ * si,
		sΩ*cω + cΩ*sω*ci, cΩ*cω*ci - sΩ*sω, -1 * cΩ * si,
		sω * si, cω * si, ci})
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
