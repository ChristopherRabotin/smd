package dynamics

import (
	"fmt"
	"math"
	"testing"
)

var (
	shortMRP = MRP{-0.074243348654559, 0.103306024060508, 0.057347988603058}
	longMRP  = MRP{3.81263, -5.30509, -2.945}
)

func TestMRPNorm(t *testing.T) {
	mrp := MRP{3, 0, 0}
	if n := mrp.norm(); n != 3 {
		t.Fatalf("norm incorrectly computed to %4.5f", n)
	}
}

func TestMRPShadow(t *testing.T) {
	if !shortMRP.Equals(&longMRP) {
		t.Fatal("short and long MRPs are not equal.")
	}
}

func TestMRPTilde(t *testing.T) {
	sV := MRP{-0.295067, 0.410571, 0.227921}
	sT := sV.Tilde(1)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if math.Abs(sT.At(j, i)+sT.At(i, j)) > 1e-12 {
				t.Fatalf("%2.6f =-? %2.6f\n", sT.At(j, i), sT.At(i, j))
			}
		}
	}
	sTExp := [][]float64{{0, -0.227921, 0.410571},
		{0.227921, 0, 0.295067},
		{-0.410571, -0.295067, 0}}
	for i, row := range sTExp {
		for j, val := range row {
			if diff := math.Abs(sT.At(i, j) - val); diff > 1e-12 {
				t.Fatalf("~(%d, %d) = %2.6f diff = %2.6f\n", i, j, sT.At(i, j), diff)
			}
		}
	}
	// Try multiplication factor.
	sT = sV.Tilde(2)
	for i, row := range sTExp {
		for j, val := range row {
			if diff := math.Abs(sT.At(i, j) - val*2); diff > 1e-12 {
				t.Fatalf("2*~(%d, %d) = %2.6f diff = %2.6f\n", i, j, sT.At(i, j), diff)
			}
		}
	}
}

func TestMRPB(t *testing.T) {
	sB := shortMRP.B()
	sExpected := [][]float64{{0.991551148415436, -0.130035547530997, 0.198096634696028},
		{0.099356406881235, 1.001871267990931, 0.160335482690017},
		{-0.215127461546006, -0.136637911928220, 0.987104582370184}}
	for i, row := range sExpected {
		for j, val := range row {
			fmt.Printf("B(%d, %d) = %2.6f diff = %2.6f\n", i, j, sB.At(j, i), math.Abs(sB.At(j, i)-val))
		}
	}
}
