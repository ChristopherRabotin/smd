package smd

import (
	"testing"
	"time"

	"github.com/gonum/matrix/mat64"
	"github.com/soniakeys/meeus/julian"
)

// The Hohmann tests are in waypoint_test.go

func TestLambertVallado(t *testing.T) {
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

func TestLambertDavisEarth2Venus(t *testing.T) {
	// TODO: This is broken.
	// These tests are from Dr. Davis' ASEN 6008 IMD course at CU.
	dt := julian.JDToTime(2455450)
	dtArr := julian.JDToTime(2455610)
	// 9.790329336673688E-01 -2.159606708797369E-01  1.964597339569911E-05
	rEarthJPL := []float64{9.790329336673688E-01 * AU, -2.159606708797369E-01 * AU, 1.964597339569911E-05 * AU}
	t.Logf("%s\t%s\t%s", dt, dtArr, Earth.HelioOrbit(dt))
	rEarth, vEarth := Earth.HelioOrbit(dt).RV()
	rVenus, vVenus := Venus.HelioOrbit(dtArr).RV()
	Ri := mat64.NewVector(3, []float64{147084764.9, -32521189.65, 467.1900914})
	Rf := mat64.NewVector(3, []float64{-88002509.16, -62680223.13, 4220331.525})
	t.Logf("\n%+v\n%+v\n%+v\n\n%+v\n%+v\n", rEarthJPL, rEarth, Ri, rVenus, Rf)
	Vi, Vf, ψ, err := Lambert(Ri, Rf, dtArr.Sub(dt), TTypeAuto, Sun)
	if err != nil {
		t.Fatalf("err = %s", err)
	}
	t.Logf("\nVi=%+v\nVf=%+v\nψ=%f", Vi, Vf, ψ)
	VinfDep := mat64.NewVector(3, nil)
	VinfArr := mat64.NewVector(3, nil)
	VinfDep.SubVec(mat64.NewVector(3, vEarth), Vi)
	VinfArr.SubVec(mat64.NewVector(3, vVenus), Vf)
	t.Logf("\nVinfDep=%+v\nVinArr=%+v", mat64.Formatted(VinfDep), mat64.Formatted(VinfArr))
}
