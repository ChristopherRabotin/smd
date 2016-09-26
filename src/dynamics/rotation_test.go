package dynamics

import (
	"math"
	"testing"
)

func TestR1R2R3(t *testing.T) {
	x := math.Pi / 3.0
	s, c := math.Sincos(x)
	r1 := R1(x)
	r2 := R2(x)
	r3 := R3(x)
	// Test items equal to 1.
	if r1.At(0, 0) != r2.At(1, 1) || r1.At(0, 0) != r3.At(2, 2) || r3.At(2, 2) != 1 {
		t.Fatal("expected R1.At(0, 0) = R2.At(1, 1) = R3.At(2, 2) = 1\n")
	}
	// Test items equal to 0.
	if r1.At(0, 1) != r1.At(0, 2) || r1.At(1, 0) != r1.At(2, 0) || r1.At(0, 1) != 0 {
		t.Fatal("misplaced zeros in R1\n")
	}
	if r2.At(0, 1) != r2.At(1, 2) || r2.At(1, 0) != r2.At(1, 2) || r2.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R2\n")
	}
	if r3.At(2, 0) != r3.At(2, 1) || r3.At(0, 2) != r3.At(1, 2) || r3.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R3\n")
	}
	// Test R1.
	if r1.At(1, 1) != r1.At(2, 2) || r1.At(2, 2) != c {
		t.Fatal("expected R1 cosines misplaced\n")
	}
	if r1.At(2, 1) != -r1.At(1, 2) || r1.At(1, 2) != s {
		t.Fatal("expected R1 sines misplaced\n")
	}
	// Test R2.
	if r2.At(0, 0) != r2.At(2, 2) || r2.At(2, 2) != c {
		t.Fatal("expected R2 cosines misplaced\n")
	}
	if r2.At(2, 0) != -r2.At(0, 2) || r2.At(2, 0) != s {
		t.Fatal("expected R2 sines misplaced\n")
	}
	// Test R3.
	if r3.At(1, 1) != r3.At(0, 0) || r3.At(0, 0) != c {
		t.Fatal("expected R3 cosines misplaced\n")
	}
	if r3.At(0, 1) != -r3.At(1, 0) || r3.At(0, 1) != s {
		t.Fatal("expected R3 sines misplaced\n")
	}
}
