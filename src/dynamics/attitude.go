package dynamics

import (
	"fmt"
	"math"

	"github.com/gonum/matrix/mat64"
)

/*-----*/
/* Modified Rodrigez Parameters */
/*-----*/

// MRP defines Modified Rodrigez Parameters.
type MRP struct {
	s1, s2, s3 float64
}

func (s *MRP) squared() float64 {
	return s.s1*s.s1 + s.s2*s.s2 + s.s3*s.s3
}

func (s *MRP) norm() float64 {
	return math.Sqrt(s.squared())
}

// Short refreshes this MRP representation to use its short notation.
func (s *MRP) Short() {
	norm := s.norm()
	if norm > 1 {
		// Switch to shadow set.
		s.s1 = -s.s1 / s.squared()
		s.s2 = -s.s2 / s.squared()
		s.s3 = -s.s3 / s.squared()
	}
}

// Tilde returns the tilde matrix of this MRP.
// The m parameter allows to multiply directly the Tilde matrix.
func (s *MRP) Tilde(m float64) *mat64.Dense {
	return mat64.NewDense(3, 3, []float64{0, -s.s3 * m, s.s2 * m,
		s.s3 * m, 0, -s.s1 * m,
		-s.s2 * m, s.s3 * m, 0})
}

// OuterProduct returns the outer product of this MRP with itself.
// The m parameter allows to multiply directly the outer product with a scalar.
func (s *MRP) OuterProduct(m float64) *mat64.Dense {
	return mat64.NewDense(3, 3, []float64{
		s.s1 * s.s1, s.s1 * s.s2, s.s1 * s.s3,
		s.s2 * s.s1, s.s2 * s.s2, s.s2 * s.s3,
		s.s3 * s.s1, s.s3 * s.s2, s.s3 * s.s3,
	})
}

// B returns the B matrix for MRP computations.
func (s *MRP) B() *mat64.Dense {
	B := mat64.NewDense(3, 3, nil)
	e1 := mat64.NewDense(3, 3, []float64{1 - s.squared(), 0, 0,
		0, 1 - s.squared(), 0,
		0, 0, 1 - s.squared()})
	e2 := s.Tilde(2)
	B.Add(e1, e2)
	B.Add(B, s.OuterProduct(2))
	return B
}

// Attitude defines an attitude with an orientation, an angular velocity and an inertial tensor.
type Attitude struct {
	Attitude      *MRP
	Velocity      *mat64.Vector
	InertiaTensor *mat64.Dense
}

// NewAttitude returns an Attitude pointer.
func NewAttitude(sigma [3]float64, omega [3]float64, tensor []float64) *Attitude {
	a := Attitude{}
	a.Attitude = &MRP{sigma[0], sigma[1], sigma[2]}
	a.Velocity = mat64.NewVector(3, []float64{omega[0], omega[1], omega[2]})
	a.InertiaTensor = mat64.NewDense(3, 3, tensor)
	return &a
}

// State returns the state of this attitude for the EOM as defined below.
func (a *Attitude) State() []float64 {
	return []float64{a.Attitude.s1, a.Attitude.s2, a.Attitude.s3, a.Velocity.At(0, 0), a.Velocity.At(1, 0), a.Velocity.At(2, 0)}
}

// EulerEOM returns the EOM for this given Attitude object.
func (a *Attitude) EulerEOM() func(t float64, state, f []float64) {
	// Let's define the multiplication factors due to the inertial tensor.
	mf1 := (a.InertiaTensor.At(1, 1) - a.InertiaTensor.At(2, 2)) / a.InertiaTensor.At(0, 0)
	mf2 := (a.InertiaTensor.At(2, 2) - a.InertiaTensor.At(0, 0)) / a.InertiaTensor.At(1, 1)
	mf3 := (a.InertiaTensor.At(0, 0) - a.InertiaTensor.At(1, 1)) / a.InertiaTensor.At(2, 2)
	fmt.Printf("mf1=%v\tmf2=%v\tmf3=%v\n", mf1, mf2, mf3)
	return func(t float64, state, f []float64) {
		// Let's create the Omega vector using BLAS.
		fmt.Printf("state = %+v\n", state)
		sigma := MRP{state[0], state[1], state[2]}
		omega := mat64.NewVector(3, []float64{state[3], state[4], state[5]})
		omega.MulVec(sigma.B(), omega)
		f[0] = 0.25 * omega.At(0, 0)
		f[1] = 0.25 * omega.At(1, 0)
		f[2] = 0.25 * omega.At(2, 0)
		f[3] = mf1 * omega.At(1, 0) * omega.At(2, 0)
		f[4] = mf2 * omega.At(0, 0) * omega.At(2, 0)
		f[5] = mf3 * omega.At(1, 0) * omega.At(0, 0)
		fmt.Printf("f = %+v\n", f)
	}
}
