package smd

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
	for i := 0.0; i <= 360; i += 0.5 {
		// Specific tests
		mi := math.Mod(i, 180)
		var expPi float64
		specificCase := false
		switch mi {
		case 0:
			specificCase = true
			expPi = 0
			break
		case 30:
			specificCase = true
			expPi = 1 / 6.
			break
		case 60:
			specificCase = true
			expPi = 1 / 3.
			break
		case 90:
			specificCase = true
			expPi = 1 / 2.
			break
		case 120:
			specificCase = true
			expPi = 2 / 3.
			break
		case 150:
			specificCase = true
			expPi = 5 / 6.
			break
		}
		if specificCase {
			if i >= 180 && i < 360 {
				expPi++
			}
			if !floats.EqualWithinAbs(Deg2rad(i)/math.Pi, expPi, 1e-10) {
				t.Fatalf("%f deg %f rad %f exp=%f", mi, Deg2rad(i)/math.Pi, Rad2deg(Deg2rad(i)), expPi)
			}
		}

		if ok, _ := anglesEqual(i, Rad2deg(Deg2rad(i))); i < 360 && !ok {
			t.Fatalf("incorrect conversion for %3.2f", i)
		} else if i == 360 && Rad2deg(Deg2rad(i)) != 0 {
			t.Fatalf("incorrect conversion for %3.2f", i)
		}
	}
	if ok, _ := anglesEqual(1, Rad2deg(Deg2rad(-359.))); !ok {
		t.Fatal("incorrect conversion for -359")
	}
	if ok, _ := anglesEqual(180, Rad2deg(Deg2rad(-180.))); !ok {
		t.Fatal("incorrect conversion for -180")
	}
	if ok, _ := anglesEqual(math.Pi/3, Deg2rad(Rad2deg(-5*math.Pi/3))); !ok {
		t.Fatal("incorrect conversion for -pi/3")
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
	if vectorsEqual([]float64{1, 0}, []float64{1, 0, 0}) {
		t.Fatal("vectors of different sizes should not be equal")
	}
	if sign(10) != 1 {
		t.Fatal("sign of 10 != 1")
	}
	if sign(-10) != -1 {
		t.Fatal("sign of -10 != 1")
	}
	if sign(0) != 1 {
		t.Fatal("sign of 0 != 1")
	}
	nilVec := []float64{0, 0, 0}
	if norm(nilVec) != 0 {
		t.Fatal("norm of a nil vector was not nil")
	}
	five0 := []float64{5, 6, 7}
	five1 := []float64{7, 6, 5}
	five2 := []float64{6, 7, 5}
	if norm(five0) != math.Sqrt(110) || norm(five0) != norm(five1) || norm(five0) != norm(five2) {
		t.Fatal("norm of the [5, 6, 7] and permutations is invalid")
	}
	uNilVec := unit(nilVec)
	for i := 0; i < 3; i++ {
		if uNilVec[i] != nilVec[i] {
			t.Fatalf("%f != %f @ i=%d", uNilVec[i], nilVec[i], i)
		}
	}
}
