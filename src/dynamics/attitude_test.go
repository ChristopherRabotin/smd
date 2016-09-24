package dynamics

import (
	"math"
	"testing"
)

var (
	ε        = 1e-12
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
			if math.Abs(sT.At(j, i)+sT.At(i, j)) > ε {
				t.Fatalf("%2.6f =-? %2.6f\n", sT.At(j, i), sT.At(i, j))
			}
		}
	}
	sTExp := [][]float64{{0, -0.227921, 0.410571},
		{0.227921, 0, 0.295067},
		{-0.410571, -0.295067, 0}}
	for i, row := range sTExp {
		for j, val := range row {
			if diff := math.Abs(sT.At(i, j) - val); diff > ε {
				t.Fatalf("~(%d, %d) = %2.6f diff = %2.6f\n", i, j, sT.At(i, j), diff)
			}
		}
	}
	// Try multiplication factor.
	sT = sV.Tilde(2)
	for i, row := range sTExp {
		for j, val := range row {
			if diff := math.Abs(sT.At(i, j) - val*2); diff > ε {
				t.Fatalf("2*~(%d, %d) = %2.6f diff = %2.6f\n", i, j, sT.At(i, j), diff)
			}
		}
	}
}

func TestOuterPoductMRP(t *testing.T) {
	oT := shortMRP.OuterProduct(1)
	oEx := [][]float64{{0.005512074819442, -0.007669785162441, -0.004257706712495},
		{-0.007669785162441, 0.010672134607190, 0.005924392690449},
		{-0.004257706712495, 0.005924392690449, 0.003288791796816}}
	for i, row := range oEx {
		for j, val := range row {
			if diff := math.Abs(oT.At(i, j) - val); diff > ε {
				t.Fatalf("OuterProduct(%d, %d) = %2.6f diff = %2.6f\n", i, j, oT.At(i, j), diff)
			}
		}
	}
	// Test multiplication factor.
	oT = shortMRP.OuterProduct(2)
	for i, row := range oEx {
		for j, val := range row {
			if diff := math.Abs(oT.At(i, j) - 2*val); diff > ε {
				t.Fatalf("OuterProduct(%d, %d) = %2.6f diff = %2.6f\n", i, j, oT.At(i, j), diff)
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
			if diff := math.Abs(sB.At(i, j) - val); diff > ε {
				t.Fatalf("B(%d, %d) = %2.6f diff = %2.6f\n", i, j, sB.At(i, j), math.Abs(sB.At(i, j)-val))
			}
		}
	}
}

func TestMomentum(t *testing.T) {
	att := NewAttitude([3]float64{0.3, -0.4, 0.5}, [3]float64{0.1, 0.4, -0.2},
		[]float64{10, 0, 0, 0, 5, 0, 0, 0, 2})
	mom := att.Momentum()
	if diff := math.Abs(mom - 2.271563338320109); diff > ε {
		t.Fatalf("angular momentum = %2.6f; diff = %2.6f\n", mom, diff)
	}
}
