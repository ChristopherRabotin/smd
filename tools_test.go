package smd

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/floats"
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
	t.Skip("XXX: The Lambert solver breaks on this case.")
	// These tests are from Dr. Davis' ASEN 6008 IMD course at CU.
	dt := julian.JDToTime(2455450)
	dtArr := julian.JDToTime(2455610)
	// 9.790329336673688E-01 -2.159606708797369E-01  1.964597339569911E-05
	rEarthJPL := []float64{9.790329336673688E-01 * AU, -2.159606708797369E-01 * AU, 1.964597339569911E-05 * AU}
	t.Logf("%s\t%s\t%s", dt, dtArr, Earth.HelioOrbit(dt))
	rEarth, vEarth := Earth.HelioOrbit(dt).RV()
	rVenus, vVenus := Venus.HelioOrbit(dtArr).RV()
	t.Logf("===V===\n%+v\n%+v\n\n", vEarth, vVenus)
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
	VinfDep.SubVec(Vi, mat64.NewVector(3, vEarth))
	VinfArr.SubVec(Vf, mat64.NewVector(3, vVenus))
	t.Logf("\nVinfDep=%+v\nVinArr=%+v", mat64.Formatted(VinfDep), mat64.Formatted(VinfArr))
}

func TestLambertDavisMars2Jupiter(t *testing.T) {
	// These tests are from Dr. Davis' ASEN 6008 IMD course at CU.
	dtDep := julian.JDToTime(2456300)
	dtArr := julian.JDToTime(2457500)
	vMars := Mars.HelioOrbit(dtDep).V()
	vJupiter := Jupiter.HelioOrbit(dtArr).V()
	Ri := mat64.NewVector(3, []float64{170145121.3, -117637192.8, -6642044.272})
	Rf := mat64.NewVector(3, []float64{-803451694.7, 121525767.1, 17465211.78})
	Vi, Vf, ψ, err := Lambert(Ri, Rf, dtArr.Sub(dtDep), TTypeAuto, Sun)
	if err != nil {
		t.Fatalf("err = %s", err)
	}
	ViExp := mat64.NewVector(3, []float64{13.74077736, 28.83099312, 0.691285008})
	VfExp := mat64.NewVector(3, []float64{-0.883933069, -7.983627014, -0.2407705978})
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

	VinfDep := mat64.NewVector(3, nil)
	VinfArr := mat64.NewVector(3, nil)
	VinfDep.SubVec(Vi, mat64.NewVector(3, vMars))
	VinfArr.SubVec(Vf, mat64.NewVector(3, vJupiter))
	vInf := mat64.Norm(VinfArr, 2)
	c3 := math.Pow(mat64.Norm(VinfDep, 2), 2)
	if !floats.EqualWithinAbs(c3, 53.59, 1e-1) {
		t.Fatalf("c3=%f expected ~53.59 km^2/s^2", c3)
	}
	if !floats.EqualWithinAbs(vInf, 4.500, 1e-2) {
		t.Fatalf("vInf=%f expected ~4.5 km/s", vInf)
	}
}
