package tools

import (
	"testing"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

func TestLambert(t *testing.T) {
	// From Vallado 4th edition, page 497
	Ri := mat64.NewVector(3, []float64{15945.34, 0, 0})
	Rf := mat64.NewVector(3, []float64{12214.83899, 10249.46731, 0})
	ViExp := mat64.NewVector(3, []float64{2.058913, 2.915965, 0})
	VfExp := mat64.NewVector(3, []float64{-3.451565, 0.910315, 0})
	for _, dm := range []float64{0, 1} {
		Vi, Vf, ψ, err := Lambert(Ri, Rf, 76.0*60, dm, smd.Earth)
		if err != nil {
			t.Fatalf("err %s", err)
		}
		if !mat64.EqualApprox(Vi, ViExp, 1e-6) {
			t.Logf("ψ=%f", ψ)
			t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vi.T()), mat64.Formatted(ViExp.T()))
			t.Fatalf("[dm=%f] incorrect Vi computed", dm)
		}
		if !mat64.EqualApprox(Vf, VfExp, 1e-6) {
			t.Logf("ψ=%f", ψ)
			t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vf.T()), mat64.Formatted(VfExp.T()))
			t.Fatalf("[dm=%f] incorrect Vf computed", dm)
		}
	}
	// Test with dm=-1
	ViExp = mat64.NewVector(3, []float64{-3.811158, -2.003854, 0})
	VfExp = mat64.NewVector(3, []float64{4.207569, 0.914724, 0})

	Vi, Vf, ψ, err := Lambert(Ri, Rf, 76.0*60, -1, smd.Earth)
	if err != nil {
		t.Fatalf("err %s", err)
	}
	if !mat64.EqualApprox(Vi, ViExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vi.T()), mat64.Formatted(ViExp.T()))
		t.Fatal("[dm=-1] incorrect Vi computed")
	}
	if !mat64.EqualApprox(Vf, VfExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vf.T()), mat64.Formatted(VfExp.T()))
		t.Fatal("[dm=-1] incorrect Vf computed")
	}
}

func TestLambertErrors(t *testing.T) {
	// Invalid R vectors
	Ri := mat64.NewVector(3, []float64{15945.34, 0, 0})
	Rf := mat64.NewVector(3, []float64{12214.83899, 10249.46731, 0})
	_, _, _, err := Lambert(Ri, Rf, 76.0*60, 2, smd.Earth)
	if err == nil {
		t.Fatal("err should not be nil if dm == 2")
	}
	_, _, _, err = Lambert(mat64.NewVector(2, []float64{15945.34, 0}), Rf, 76.0*60, 2, smd.Earth)
	if err == nil {
		t.Fatal("err should not be nil if the R vectors are of different dimensions")
	}
	_, _, _, err = Lambert(mat64.NewVector(2, []float64{15945.34, 0}), mat64.NewVector(2, []float64{12214.83899, 10249.46731}), 76.0*60, 2, smd.Earth)
	if err == nil {
		t.Fatal("err should not be nil if the R vectors are of not of dimension 3x1")
	}
}
