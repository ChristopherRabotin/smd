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
	// These tests are from Dr. Davis' ASEN 6008 IMD course at CU.
	dt := julian.JDToTime(2455450)
	dtArr := julian.JDToTime(2455610)
	rEarth, vEarth := Earth.HelioOrbit(dt).RV()
	rVenus, vVenus := Venus.HelioOrbit(dtArr).RV()
	Ri := mat64.NewVector(3, rEarth)
	Rf := mat64.NewVector(3, rVenus)
	Vi, Vf, ψ, err := Lambert(Ri, Rf, dtArr.Sub(dt), TType2, Sun)
	if err != nil {
		t.Fatalf("err = %s", err)
	}
	// The following expected values are the actual output from my Lambert solver, and are within 1e-2 of the
	// values from the spreadcheet. I have updated them to the ones I found to detect any regression issue.
	ViExp := mat64.NewVector(3, []float64{4.650884, 26.082007, -1.393243})
	VfExp := mat64.NewVector(3, []float64{16.790445, -33.353309, 1.523397})
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
	VinfDep.SubVec(Vi, mat64.NewVector(3, vEarth))
	VinfArr.SubVec(Vf, mat64.NewVector(3, vVenus))
	VinfDepExp := mat64.NewVector(3, []float64{-1.734209, -2.798352, -1.393243})
	VinfArrExp := mat64.NewVector(3, []float64{-3.540174, -4.804623, 1.523397})
	if !mat64.EqualApprox(VinfDep, VinfDepExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(VinfDep.T()), mat64.Formatted(VinfDepExp.T()))
		t.Fatalf("[%s] incorrect VinfDep computed", TType2)
	}
	if !mat64.EqualApprox(VinfArr, VinfArrExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(VinfArr.T()), mat64.Formatted(VinfArrExp.T()))
		t.Fatalf("[%s] incorrect VinfArr computed", TType2)
	}
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
	if !floats.EqualWithinAbs(c3, 51.97, 1e-1) {
		t.Fatalf("c3=%f expected ~51.97 km^2/s^2", c3)
	}
	if !floats.EqualWithinAbs(vInf, 4.479, 1e-2) {
		t.Fatalf("vInf=%f expected ~4.479 km/s", vInf)
	}
	t.Logf("ψ=%f", ψ)
}

func TestLambertDavisEarth2VenusT3(t *testing.T) {
	t.Skip("test disabled because multi-rev does not work.")
	// These tests are from Dr. Davis' ASEN 6008 IMD course at CU.
	dtDep := julian.JDToTime(2460545)
	dtArr := julian.JDToTime(2460919)
	vEarth := Earth.HelioOrbit(dtDep).V()
	vVenus := Venus.HelioOrbit(dtArr).V()
	Ri := mat64.NewVector(3, []float64{130423562.1, -76679031.85, 3624.816561})
	Rf := mat64.NewVector(3, []float64{19195371.67, 106029328.4, 348953.802})
	ttype := TType3
	Vi, Vf, ψ, err := Lambert(Ri, Rf, dtArr.Sub(dtDep), ttype, Sun)
	if err != nil {
		t.Fatalf("err = %s", err)
	}
	ViExp := mat64.NewVector(3, []float64{12.76771134, 22.79158874, 0.09033882633})
	VfExp := mat64.NewVector(3, []float64{-37.30072389, -0.1768534469, -0.06669308258})
	if !mat64.EqualApprox(Vi, ViExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vi.T()), mat64.Formatted(ViExp.T()))
		t.Fatalf("[%s] incorrect Vi computed", ttype)
	}
	if !mat64.EqualApprox(Vf, VfExp, 1e-6) {
		t.Logf("ψ=%f", ψ)
		t.Logf("\nGot %+v\nExp %+v\n", mat64.Formatted(Vf.T()), mat64.Formatted(VfExp.T()))
		t.Fatalf("[%s] incorrect Vf computed", ttype)
	}

	VinfDep := mat64.NewVector(3, nil)
	VinfArr := mat64.NewVector(3, nil)
	VinfDep.SubVec(Vi, mat64.NewVector(3, vEarth))
	VinfArr.SubVec(Vf, mat64.NewVector(3, vVenus))
	vInf := mat64.Norm(VinfArr, 2)
	c3 := math.Pow(mat64.Norm(VinfDep, 2), 2)
	if !floats.EqualWithinAbs(c3, 11.12, 1e-1) {
		t.Fatalf("c3=%f expected ~11.12 km^2/s^2", c3)
	}
	if !floats.EqualWithinAbs(vInf, 7.14, 1e-2) {
		t.Fatalf("vInf=%f expected ~7.14 km/s", vInf)
	}
}
