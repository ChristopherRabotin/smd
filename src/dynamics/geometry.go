package dynamics

import (
	"math"

	"github.com/gonum/matrix/mat64"
)

// norm returns the norm of a given vector which is supposed to be 3x1.
func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// dot performs the inner product.
func dot(a, b []float64) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

// deg2rad converts degress to radians.
func deg2rad(a float64) float64 {
	return a / 180.0 * 2 * math.Pi
}

func rad2deg(a float64) float64 {
	return a / (2 * math.Pi) * 180.0
}

// MxV33 multiplies a matrix with a vector. Note that there is no dimension check!
func MxV33(m *mat64.Dense, v []float64) (o []float64) {
	o = make([]float64, 3)
	o[0] = m.At(0, 0)*v[0] + m.At(0, 1)*v[1] + m.At(0, 2)*v[2]
	o[1] = m.At(1, 0)*v[0] + m.At(1, 1)*v[1] + m.At(1, 2)*v[2]
	o[2] = m.At(2, 0)*v[0] + m.At(2, 1)*v[1] + m.At(2, 2)*v[2]
	return
}
