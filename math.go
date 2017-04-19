package smd

import (
	"math"

	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

const (
	deg2rad = math.Pi / 180
	rad2deg = 1 / deg2rad
)

// Norm returns the Norm of a given vector which is supposed to be 3x1.
func Norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// Unit returns the Unit vector of a given vector.
func Unit(a []float64) (b []float64) {
	n := Norm(a)
	if floats.EqualWithinAbs(n, 0, 1e-12) {
		return []float64{0, 0, 0}
	}
	b = make([]float64, len(a))
	for i, val := range a {
		b[i] = val / n
	}
	return
}

// unitVec returns the unit vector of a given mat64.Vector.
func unitVec(a *mat64.Vector) (b *mat64.Vector) {
	b = mat64.NewVector(a.Len(), nil)
	n := mat64.Norm(a, 2)
	if floats.EqualWithinAbs(n, 0, 1e-12) {
		return // Nil vector
	}
	b.ScaleVec(1/n, a)
	return
}

// Sign returns the Sign of a given number.
func Sign(v float64) float64 {
	if floats.EqualWithinAbs(v, 0, 1e-12) {
		return 1
	}
	return v / math.Abs(v)
}

// Dot performs the inner product via mat64/BLAS.
func Dot(a, b []float64) float64 {
	return mat64.Dot(mat64.NewVector(len(a), a), mat64.NewVector(len(b), b))
}

// dot performs the inner product via mat64/BLAS.
func dot(a, b []float64) float64 {
	rtn := 0.
	for i := 0; i < len(a); i++ {
		rtn += a[i] * b[i]
	}
	return rtn
}

// Cross performs the Cross product.
func Cross(a, b []float64) []float64 {
	return []float64{a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0]} // Cross product R x V.
}

// cross performs the cross product from two mat64.Vectors.
func crossVec(a, b *mat64.Vector) *mat64.Vector {
	rslt := mat64.NewVector(3, nil) // only support dim 3 (cross only defined in dims 3 and 7)
	rslt.SetVec(0, a.At(1, 0)*b.At(2, 0)-a.At(2, 0)*b.At(1, 0))
	rslt.SetVec(1, a.At(2, 0)*b.At(0, 0)-a.At(0, 0)*b.At(2, 0))
	rslt.SetVec(2, a.At(0, 0)*b.At(1, 0)-a.At(1, 0)*b.At(0, 0))
	return rslt
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
	if Norm(a) == 0 {
		return []float64{0, 0, 0}
	}
	b[0] = Norm(a)
	b[1] = math.Acos(a[2] / b[0])
	b[2] = math.Atan2(a[1], a[0])
	return
}

// Deg2rad converts degrees to radians, and enforced only positive numbers.
func Deg2rad(a float64) float64 {
	if a < 0 {
		a += 360
	}
	return math.Mod(a*deg2rad, 2*math.Pi)
}

// Rad2deg converts radians to degrees, and enforced only positive numbers.
func Rad2deg(a float64) float64 {
	if a < 0 {
		a += 2 * math.Pi
	}
	return math.Mod(a/deg2rad, 360)
}

// Rad2deg180 converts radians to degrees, and enforce between +/-180.
func Rad2deg180(a float64) float64 {
	if a < -math.Pi {
		a += 2 * math.Pi
	} else if a > math.Pi {
		a -= 2 * math.Pi
	}
	return math.Mod(a/deg2rad, 360)
}

// DenseIdentity returns an identity matrix of type Dense and of the provided size.
func DenseIdentity(n int) *mat64.Dense {
	return ScaledDenseIdentity(n, 1)
}

// ScaledDenseIdentity returns an identity matrix time of type Dense a scaling factor of the provided size.
func ScaledDenseIdentity(n int, s float64) *mat64.Dense {
	vals := make([]float64, n*n)
	for j := 0; j < n*n; j++ {
		if j%(n+1) == 0 {
			vals[j] = s
		} else {
			vals[j] = 0
		}
	}
	return mat64.NewDense(n, n, vals)
}
