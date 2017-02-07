package smd

import (
	"testing"
	"time"

	"github.com/gonum/matrix/mat64"
)

// The Hohmann tests are in waypoint_test.go

func TestLambert(t *testing.T) {
	// From Vallado 4th edition, page 497
	Ri := mat64.NewVector(3, []float64{15945.34, 0, 0})
	Rf := mat64.NewVector(3, []float64{12214.83899, 10249.46731, 0})
	ViExp := mat64.NewVector(3, []float64{2.058913, 2.915965, 0})
	VfExp := mat64.NewVector(3, []float64{-3.451565, 0.910315, 0})
	for _, dm := range []TransferType{TTypeAuto, TType1} {
		Vi, Vf, ψ, err := Lambert(Ri, Rf, 76.0*time.Minute, dm, Earth)
		if err != nil {
			t.Fatalf("err %s", err)
		}
		if !mat64.EqualApprox(Vi, ViExp, 1e-6) {
			t.Logf("ψ=%f", ψ)
			t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vi.T()), mat64.Formatted(ViExp.T()))
			t.Fatalf("[%s] incorrect Vi computed", dm)
		}
		if !mat64.EqualApprox(Vf, VfExp, 1e-6) {
			t.Logf("ψ=%f", ψ)
			t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vf.T()), mat64.Formatted(VfExp.T()))
			t.Fatalf("[%s] incorrect Vf computed", dm)
		}
		t.Logf("[OK] %s", dm)
	}
	// Test with dm=-1
	ViExp = mat64.NewVector(3, []float64{-3.811158, -2.003854, 0})
	VfExp = mat64.NewVector(3, []float64{4.207569, 0.914724, 0})

	Vi, Vf, ψ, err := Lambert(Ri, Rf, 76.0*time.Minute, TType2, Earth)
	if err != nil {
		t.Fatalf("err %s", err)
	}
	if !mat64.EqualApprox(Vi, ViExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vi.T()), mat64.Formatted(ViExp.T()))
		t.Fatalf("[%s] incorrect Vi computed", TType2)
	}
	if !mat64.EqualApprox(Vf, VfExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vf.T()), mat64.Formatted(VfExp.T()))
		t.Fatalf("[%s] incorrect Vf computed", TType2)
	}
	t.Logf("[OK] %s", TType2)
}

func TestLambertErrors(t *testing.T) {
	// Invalid R vectors
	Rf := mat64.NewVector(3, []float64{12214.83899, 10249.46731, 0})
	_, _, _, err := Lambert(mat64.NewVector(2, []float64{15945.34, 0}), Rf, 76.0*time.Minute, 2, Earth)
	if err == nil {
		t.Fatal("err should not be nil if the R vectors are of different dimensions")
	}
	_, _, _, err = Lambert(mat64.NewVector(2, []float64{15945.34, 0}), mat64.NewVector(2, []float64{12214.83899, 10249.46731}), 76.0*time.Minute, 2, Earth)
	if err == nil {
		t.Fatal("err should not be nil if the R vectors are of not of dimension 3x1")
	}
}
