package dynamics

import (
	"math"

	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// norm returns the norm of a given vector which is supposed to be 3x1.
func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// unit returns the unit vector of a given vector.
func unit(a []float64) (b []float64) {
	n := norm(a)
	if floats.EqualWithinAbs(n, 0, 1e-12) {
		return []float64{0, 0, 0}
	}
	b = make([]float64, len(a))
	for i, val := range a {
		b[i] = val / n
	}
	return
}

// sign returns the sign of a given number.
func sign(v float64) float64 {
	if floats.EqualWithinAbs(v, 0, 1e-12) {
		return 1
	}
	return v / math.Abs(v)
}

// dot performs the inner product via mat64/BLAS.
func dot(a, b []float64) float64 {
	return mat64.Dot(mat64.NewVector(len(a), a), mat64.NewVector(len(b), b))
}

// cross performs the inner product.
func cross(a, b []float64) []float64 {
	return []float64{a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0]} // Cross product R x V.
}

// Spherical2Cartesian returns the provided spherical coordinates vector in Cartesian.
func Spherical2Cartesian(a []float64) (b []float64) {
	b = make([]float64, 3)
	sθ, cθ := math.Sincos(a[1])
	sφ, cφ := math.Sincos(a[2])
	b[0] = a[0] * sθ * cφ
	b[1] = a[0] * sθ * sφ
	b[2] = a[0] * cθ
	return
}

// Cartesian2Spherical returns the provided Cartesian coordinates vector in spherical.
func Cartesian2Spherical(a []float64) (b []float64) {
	b = make([]float64, 3)
	if norm(a) == 0 {
		return []float64{0, 0, 0}
	}
	b[0] = norm(a)
	b[1] = math.Acos(a[2] / b[0])
	b[2] = math.Atan2(a[1], a[0])
	return
}

// Deg2rad converts degrees to radians.
func Deg2rad(a float64) float64 {
	return (a / 360.0) * 2 * math.Pi
}

// Rad2deg converts radians to degrees.
func Rad2deg(a float64) float64 {
	return a / (2 * math.Pi) * 360.0
}

// MxV33 multiplies a matrix with a vector. Note that there is no dimension check!
func MxV33(m *mat64.Dense, v []float64) (o []float64) {
	vVec := mat64.NewVector(len(v), v)
	var rVec mat64.Vector
	rVec.MulVec(m, vVec)
	return []float64{rVec.At(0, 0), rVec.At(1, 0), rVec.At(2, 0)}
}
