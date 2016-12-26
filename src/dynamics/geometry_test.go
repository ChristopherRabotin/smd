package dynamics

import (
	"math"
	"testing"

	"github.com/gonum/floats"
)

func TestCross(t *testing.T) {
	i := []float64{1, 0, 0}
	j := []float64{0, 1, 0}
	k := []float64{0, 0, 1}
	if !vectorsEqual(cross(i, j), k) {
		t.Fatal("i x j != k")
	}
	if !vectorsEqual(cross(j, k), i) {
		t.Fatal("j x k != i")
	}
	if !vectorsEqual(cross([]float64{2, 3, 4}, []float64{5, 6, 7}), []float64{-3, 6, -3}) {
		t.Fatal("cross fail")
	}
	// From Vallado
	if !vectorsEqual(cross([]float64{6524.834, 6862.875, 6448.296}, []float64{4.901327, 5.533756, -1.976341}), []float64{-4.924667792015100e4, 4.450050424118601e4, 0.246964476137900e4}) {
		t.Fatal("cross fail")
	}
}

func TestAngles(t *testing.T) {
	for i := 0.0; i < 360; i += 0.5 {
		if ok, _ := anglesEqual(i, Rad2deg(Deg2rad(i))); !ok {
			t.Fatalf("incorrect conversion for %3.2f", i)
		}
	}
}

func TestSpherical2Cartisean(t *testing.T) {
	a := make([]float64, 3)
	incr := math.Pi / 10
	for r := 0.0; r < 1000; r += 100 {
		for θ := incr; θ < math.Pi; θ += incr {
			for φ := incr; φ < 2*math.Pi; φ += incr {
				a[0] = r
				a[1] = θ
				a[2] = φ
				b := Cartesian2Spherical(Spherical2Cartesian(a))
				if r == 0.0 {
					if b[0] != 0 || b[1] != 0 || b[2] != 0 {
						t.Fatal("zero norm should return zero vector")
					}
					continue
				}
				if !floats.EqualWithinAbs(a[0], b[0], 1e-12) {
					t.Fatalf("r incorrect (%f != %f) for r=%f", a[0], b[0], r)
				}
				if ok, err := anglesEqual(a[1], b[1]); !ok {
					t.Fatalf("θ incorrect (%f != %f) %s", a[1], b[1], err)
				}
				if ok, err := anglesEqual(a[2], b[2]); !ok {
					t.Fatalf("φ incorrect (%f != %f) %s", a[2], b[2], err)
				}
			}
		}
	}
}

func TestMisc(t *testing.T) {
	if sign(10) != 1 {
		t.Fatal("sign of 10 != 1")
	}
	if sign(-10) != -1 {
		t.Fatal("sign of -10 != 1")
	}
	nilVec := []float64{0, 0, 0}
	if norm(nilVec) != 0 {
		t.Fatal("norm of a nil vector was not nil")
	}
	uNilVec := unit(nilVec)
	for i := 0; i < 3; i++ {
		if uNilVec[i] != nilVec[i] {
			t.Fatalf("%f != %f @ i=%d", uNilVec[i], nilVec[i], i)
		}
	}
}
