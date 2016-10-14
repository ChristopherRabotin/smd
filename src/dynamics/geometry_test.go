package dynamics

import (
	"fmt"
	"math"
	"testing"
)

//anglesEqual returns whether two angles in Radians are equal.
func anglesEqual(a, b float64) (bool, error) {
	diff := math.Abs(a - b)
	if diff < eps || math.Abs(diff-2*math.Pi) < eps {
		return true, nil
	}
	return false, fmt.Errorf("difference of %3.10fπ", diff/math.Pi)
}

func TestAngles(t *testing.T) {
	for i := 0.0; i < 360; i += 0.5 {
		if ok, err := floatEqual(i, Rad2deg(Deg2rad(i))); !ok {
			t.Fatalf("incorrect conversion for %3.2f, %s", i, err)
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
				if ok, err := floatEqual(a[0], b[0]); !ok {
					t.Fatalf("r incorrect (%f != %f) %s for r=%f", a[0], b[0], err, r)
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
