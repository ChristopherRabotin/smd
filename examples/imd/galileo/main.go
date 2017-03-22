package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

var (
	minRadius = 300 + smd.Earth.Radius // km
	launch    = time.Date(1989, 10, 8, 0, 0, 0, 0, time.UTC)
	vga1      = time.Date(1990, 2, 10, 0, 0, 0, 0, time.UTC)
	ega1      = time.Date(1990, 12, 10, 0, 0, 0, 0, time.UTC)
	ega2      = time.Date(1992, 12, 9, 12, 0, 0, 0, time.UTC)
	joi       = time.Date(1996, 3, 21, 12, 0, 0, 0, time.UTC)
)

func main() {
	resonance := ega2.Sub(ega1).Hours() / (365.242189 * 24)
	fmt.Printf("%s\t~%f orbits\n", ega2.Sub(ega1), resonance)

	fmt.Println("==== Launch -> VGA ====")
	// hwQ 1
	earthAtLaunch := smd.Earth.HelioOrbit(launch)
	eLaunchR := mat64.NewVector(3, earthAtLaunch.R())
	venusAtVGA := smd.Venus.HelioOrbit(vga1)
	vga1R := mat64.NewVector(3, venusAtVGA.R())
	_, VfVGA, _, _ := smd.Lambert(eLaunchR, vga1R, vga1.Sub(launch), smd.TTypeAuto, smd.Sun)
	vfloatsVGA := []float64{VfVGA.At(0, 0), VfVGA.At(1, 0), VfVGA.At(2, 0)}
	scOrbitAtVGAIn := smd.NewOrbitFromRV(venusAtVGA.R(), vfloatsVGA, smd.Sun)
	scOrbitAtVGAIn.ToXCentric(smd.Venus, vga1)

	fmt.Println("==== VGA -> EGA1 ====")
	earthAtEGA1 := smd.Earth.HelioOrbit(ega1)
	ega1R := mat64.NewVector(3, earthAtEGA1.R())
	ViVGA, VfGA1, _, _ := smd.Lambert(vga1R, ega1R, ega1.Sub(vga1), smd.TTypeAuto, smd.Sun)
	scOrbitAtVGAOut := smd.NewOrbitFromRV(venusAtVGA.R(), []float64{ViVGA.At(0, 0), ViVGA.At(1, 0), ViVGA.At(2, 0)}, smd.Sun)
	scOrbitAtVGAOut.ToXCentric(smd.Venus, vga1)
	// Okay, we have all the info for Venus, let's compute stuff.
	_, rPVenus, bTVenus, bRVenus, _, _ := smd.GAFromVinf(scOrbitAtVGAIn.V(), scOrbitAtVGAOut.V(), smd.Venus)
	fmt.Printf("==== VENUS INFO ====\nrP=%f km\tBt=%f km\tBr=%f\nVin=%f\tVout=%f\nDelta=%f\n\n", rPVenus, bTVenus, bRVenus, scOrbitAtVGAIn.VNorm(), scOrbitAtVGAOut.VNorm(), scOrbitAtVGAOut.VNorm()-scOrbitAtVGAIn.VNorm())

	scOrbitAtEGA1 := smd.NewOrbitFromRV(earthAtEGA1.R(), []float64{VfGA1.At(0, 0), VfGA1.At(1, 0), VfGA1.At(2, 0)}, smd.Sun)
	scOrbitAtEGA1.ToXCentric(smd.Earth, ega1)
	vInfInGA1 := scOrbitAtEGA1.V()
	vInfOutGA1Norm := scOrbitAtEGA1.VNorm() // Called OutGA1 because we suppose there was no maneuver during the flyby

	fmt.Println("==== EGA2 -> JOI ====")
	// hwQ 2
	earthAtEGA2 := smd.Earth.HelioOrbit(ega2)
	jupiterAtJOI := smd.Jupiter.HelioOrbit(joi)
	ega2R := mat64.NewVector(3, earthAtEGA2.R())
	joiR := mat64.NewVector(3, jupiterAtJOI.R())
	ViGA2, VfJOI, _, _ := smd.Lambert(ega2R, joiR, joi.Sub(ega2), smd.TTypeAuto, smd.Sun)
	scOrbitAtEGA2 := smd.NewOrbitFromRV(earthAtEGA2.R(), []float64{ViGA2.At(0, 0), ViGA2.At(1, 0), ViGA2.At(2, 0)}, smd.Sun)
	scOrbitAtEGA2.ToXCentric(smd.Earth, ega2)
	vInfOutGA2 := scOrbitAtEGA2.V()
	vInfOutGA2Norm := scOrbitAtEGA2.VNorm()
	scOrbitAtJOI := smd.NewOrbitFromRV(jupiterAtJOI.R(), []float64{VfJOI.At(0, 0), VfJOI.At(1, 0), VfJOI.At(2, 0)}, smd.Sun)
	scOrbitAtJOI.ToXCentric(smd.Jupiter, joi)
	Φfpa := math.Atan2(scOrbitAtJOI.SinΦfpa(), scOrbitAtJOI.CosΦfpa())
	fmt.Printf("==== JUPITER INFO ====\nOrbit %s\nPeriod: %s (~%f days)\tEnergy: %f\tΦfpa=%f\napo=%f km\tperi=%f km\n\n", scOrbitAtJOI, scOrbitAtJOI.Period(), scOrbitAtJOI.Period().Hours()/24, scOrbitAtJOI.Energyξ(), Φfpa*r2d, scOrbitAtJOI.Apoapsis(), scOrbitAtJOI.Periapsis())

	fmt.Printf("%+v\n%f km/s\n", vInfOutGA2, vInfOutGA2Norm)

	fmt.Println("==== Earth resonnance ====")
	aResonance := math.Pow(smd.Sun.GM()*math.Pow(resonance*earthAtEGA1.Period().Seconds()/(2*math.Pi), 2), 1/3.)
	VScSunNorm := math.Sqrt(smd.Sun.GM() * ((2 / earthAtEGA1.RNorm()) - 1/aResonance))
	// Compute angle theta for EGA1
	theta := math.Acos((math.Pow(VScSunNorm, 2) - math.Pow(vInfOutGA1Norm, 2) - math.Pow(earthAtEGA1.VNorm(), 2)) / (-2 * vInfOutGA1Norm * earthAtEGA1.VNorm()))
	fmt.Printf("theta = %f\n", theta*r2d)
	// Compute the VNC2ECI DCMs for EGA1 and EGA2.
	V := unit(earthAtEGA1.V())
	N := unit(earthAtEGA1.H())
	C := cross(V, N)
	dcmVal := make([]float64, 9)
	for i := 0; i < 3; i++ {
		dcmVal[i] = V[i]
		dcmVal[i+3] = N[i]
		dcmVal[i+6] = C[i]
	}
	DCM := mat64.NewDense(3, 3, dcmVal)

	ψ := 165.924 * d2r // My choice of Phi.

	sψ, cψ := math.Sincos(ψ)
	vInfOutGA1VNC := []float64{vInfOutGA1Norm * math.Cos(math.Pi-theta), vInfOutGA1Norm * math.Sin(math.Pi-theta) * cψ, -vInfOutGA1Norm * math.Sin(math.Pi-theta) * sψ}
	vInfOutGA1 := MxV33(DCM, vInfOutGA1VNC)
	_, rPEGA1, bTEGA1, bREGA1, _, _ := smd.GAFromVinf(vInfInGA1, vInfOutGA1, smd.Earth)
	fmt.Printf("==== EGA1 INFO ====\nrP=%f km\tBt=%f km\tBr=%f\nVin=%f\tVout=%f\nDelta=%f\n\n", rPEGA1, bTEGA1, bREGA1, norm(vInfInGA1), norm(vInfOutGA1), norm(vInfOutGA1)-norm(vInfInGA1))

	vInfInGA2 := make([]float64, 3)
	for i := 0; i < 3; i++ {
		vInfInGA2[i] = vInfOutGA1[i] + scOrbitAtEGA1.V()[i] - scOrbitAtEGA2.V()[i]
	}
	_, rPEGA2, bTEGA2, bREGA2, _, _ := smd.GAFromVinf(vInfOutGA1, vInfInGA2, smd.Earth)
	fmt.Printf("==== EGA2 INFO ====\nrP=%f km\tBt=%f km\tBr=%f\nVin=%f\tVout=%f\nDelta=%f\n\n", rPEGA2, bTEGA2, bREGA2, norm(vInfOutGA1), norm(vInfInGA2), norm(vInfInGA2)-norm(vInfOutGA1))

}

// Unshamefully copied from smd/math.go
func cross(a, b []float64) []float64 {
	return []float64{a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0]} // Cross product R x V.
}

// norm returns the norm of a given vector which is supposed to be 3x1.
func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// unit returns the unit vector of a given vector.
func unit(a []float64) (b []float64) {
	n := norm(a)
	if floats.EqualWithinAbs(n, 0, 1e-12) {
		return []float64{0, 0, 0}
	}
	b = make([]float64, len(a))
	for i, val := range a {
		b[i] = val / n
	}
	return
}

// MxV33 multiplies a matrix with a vector. Note that there is no dimension check!
func MxV33(m *mat64.Dense, v []float64) (o []float64) {
	vVec := mat64.NewVector(len(v), v)
	var rVec mat64.Vector
	rVec.MulVec(m, vVec)
	return []float64{rVec.At(0, 0), rVec.At(1, 0), rVec.At(2, 0)}
}
