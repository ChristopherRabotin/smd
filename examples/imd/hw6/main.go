package main

import (
	"fmt"
	"math"
	"os"
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
	launch = time.Date(1989, 10, 8, 0, 0, 0, 0, time.UTC)
	vga1   = time.Date(1990, 2, 10, 0, 0, 0, 0, time.UTC)
	ega1   = time.Date(1990, 12, 10, 0, 0, 0, 0, time.UTC)
	ega2   = time.Date(1992, 12, 9, 12, 0, 0, 0, time.UTC)
	joi    = time.Date(1996, 3, 21, 12, 0, 0, 0, time.UTC)
)

func main() {
	resonance := ega2.Sub(ega1).Hours() / (365.242189 * 24)
	fmt.Printf("%s\t~%f orbits\n", ega2.Sub(ega1), resonance)
	var ViGA2, VfGA1 *mat64.Vector

	fmt.Println("==== QUESTION 1 ====")
	// hwQ 1
	vga1R := mat64.NewVector(3, smd.Venus.HelioOrbit(vga1).R())
	earthAtEGA1 := smd.Earth.HelioOrbit(ega1)
	ega1R := mat64.NewVector(3, earthAtEGA1.R())
	_, VfGA1, _, _ = smd.Lambert(vga1R, ega1R, ega1.Sub(vga1), smd.TTypeAuto, smd.Sun)
	vfloats1 := []float64{VfGA1.At(0, 0), VfGA1.At(1, 0), VfGA1.At(2, 0)}
	ega1Orbit := smd.NewOrbitFromRV(earthAtEGA1.R(), vfloats1, smd.Sun)
	ega1Orbit.ToXCentric(smd.Earth, ega1)
	vInfInGA1 := ega1Orbit.V()
	vInfOutGA1Norm := ega1Orbit.VNorm()
	fmt.Printf("%+v\n%f km/s\n", vInfInGA1, vInfOutGA1Norm)
	fmt.Println("==== QUESTION 2 ====")
	// hwQ 2
	earthAtEGA2 := smd.Earth.HelioOrbit(ega2)
	ega2R := mat64.NewVector(3, earthAtEGA2.R())
	joiR := mat64.NewVector(3, smd.Jupiter.HelioOrbit(joi).R())
	ViGA2, _, _, _ = smd.Lambert(ega2R, joiR, joi.Sub(ega2), smd.TTypeAuto, smd.Sun)
	vfloats2 := []float64{ViGA2.At(0, 0), ViGA2.At(1, 0), ViGA2.At(2, 0)}
	ega2Orbit := smd.NewOrbitFromRV(earthAtEGA2.R(), vfloats2, smd.Sun)
	ega2Orbit.ToXCentric(smd.Earth, ega2)
	vInfOutGA2 := ega2Orbit.V()
	vInfOutGA2Norm := ega2Orbit.VNorm()
	fmt.Printf("%+v\n%f km/s\n", vInfOutGA2, vInfOutGA2Norm)

	fmt.Println("==== QUESTION 3 ====")
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
	data := "psi\trP1\trP2\n"
	step := (2 * math.Pi) / 10000
	for ψ := step; ψ < 2*math.Pi; ψ += step {
		sψ, cψ := math.Sincos(ψ)
		vInfOutGA1VNC := []float64{vInfOutGA1Norm * math.Cos(math.Pi-theta), vInfOutGA1Norm * math.Sin(math.Pi-theta) * cψ, -vInfOutGA1Norm * math.Sin(math.Pi-theta) * sψ}
		vInfOutGA1 := MxV33(DCM, vInfOutGA1VNC)
		_, rP1, _, _, _, _ := smd.GAFromVinf(vInfInGA1, vInfOutGA1, smd.Earth)

		vInfInGA2 := make([]float64, 3)
		for i := 0; i < 3; i++ {
			vInfInGA2[i] = vInfOutGA1[i] + ega1Orbit.V()[i] - ega2Orbit.V()[i]
		}
		_, rP2, _, _, _, _ := smd.GAFromVinf(vInfOutGA1, vInfInGA2, smd.Earth)
		data += fmt.Sprintf("%f\t%f\t%f\n", ψ*r2d, rP1, rP2)
	}
	// Export data
	f, err := os.Create("./q3.tsv")
	if err != nil {
		panic(err)
	}
	f.WriteString(data)
	f.Close()
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
