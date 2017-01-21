package dynamics

import (
	"fmt"
	"math"
	"testing"

	"github.com/gonum/floats"
)

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("code did not panic")
		}
	}()
	f()
}

func vectorsEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := len(a) - 1; i >= 0; i-- {
		if !floats.EqualWithinRel(a[i], b[i], 1e-3) {
			return false
		}
	}
	return true
}

//anglesEqual returns whether two angles in Radians are equal.
func anglesEqual(a, b float64) (bool, error) {
	diff := math.Mod(math.Abs(a-b), 2*math.Pi)
	if diff < angleε {
		return true, nil
	}
	return false, fmt.Errorf("difference of %3.10f degrees", math.Abs(Rad2deg(diff)))
}
