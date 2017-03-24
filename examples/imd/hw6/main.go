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
	var ViGA2, VfGA1 *mat64.Vector

	fmt.Println("==== QUESTION 1 ====")
	// hwQ 1
	vga1R := mat64.NewVector(3, smd.Venus.HelioOrbit(vga1).R())
	earthAtEGA1 := smd.Earth.HelioOrbit(ega1)
	ega1R := mat64.NewVector(3, earthAtEGA1.R())
	_, VfGA1, _, _ = smd.Lambert(vga1R, ega1R, ega1.Sub(vga1), smd.TTypeAuto, smd.Sun)
	vInfInEGA1Vec := mat64.NewVector(3, nil)
	vInfInEGA1Vec.SubVec(VfGA1, mat64.NewVector(3, earthAtEGA1.V()))
	vInfInEGA1 := []float64{vInfInEGA1Vec.At(0, 0), vInfInEGA1Vec.At(1, 0), vInfInEGA1Vec.At(2, 0)}
	vInfInEGA1Norm := norm(vInfInEGA1)
	fmt.Printf("%+v\n%f km/s\n", vInfInEGA1, vInfInEGA1Norm)
	fmt.Println("==== QUESTION 2 ====")
	// hwQ 2
	earthAtEGA2 := smd.Earth.HelioOrbit(ega2)
	ega2R := mat64.NewVector(3, earthAtEGA2.R())
	joiR := mat64.NewVector(3, smd.Jupiter.HelioOrbit(joi).R())
	ViGA2, _, _, _ = smd.Lambert(ega2R, joiR, joi.Sub(ega2), smd.TTypeAuto, smd.Sun)
	vInfOutEGA2Vec := mat64.NewVector(3, nil)
	vInfOutEGA2Vec.SubVec(ViGA2, mat64.NewVector(3, earthAtEGA2.V()))
	vInfOutEGA2 := []float64{vInfOutEGA2Vec.At(0, 0), vInfOutEGA2Vec.At(1, 0), vInfOutEGA2Vec.At(2, 0)}
	vInfOutEGA2Norm := norm(vInfInEGA1)
	fmt.Printf("%+v\n%f km/s\n", vInfOutEGA2, vInfOutEGA2Norm)

	fmt.Println("==== QUESTION 3 ====")
	aResonance := math.Pow(smd.Sun.GM()*math.Pow(resonance*earthAtEGA1.Period().Seconds()/(2*math.Pi), 2), 1/3.)
	VScSunNorm := math.Sqrt(smd.Sun.GM() * ((2 / earthAtEGA1.RNorm()) - 1/aResonance))
	// Compute angle theta for EGA1
	theta := math.Acos((math.Pow(VScSunNorm, 2) - math.Pow(vInfInEGA1Norm, 2) - math.Pow(earthAtEGA1.VNorm(), 2)) / (-2 * vInfInEGA1Norm * earthAtEGA1.VNorm()))
	fmt.Printf("theta = %f\n", theta*r2d)
	// Compute the VNC2ECI DCMs for EGA1.
	// WARNING: We are generating the transposed DCM because it's simpler code.
	V := unit(earthAtEGA1.V())
	N := unit(earthAtEGA1.H())
	C := cross(V, N)
	dcmVal := make([]float64, 9)
	for i := 0; i < 3; i++ {
		dcmVal[i] = V[i]
		dcmVal[i+3] = N[i]
		dcmVal[i+6] = C[i]
	}
	transposedDCM := mat64.NewDense(3, 3, dcmVal)
	data := "psi\trP1\trP2\n"
	step := (2 * math.Pi) / 10000
	// Print when both become higher than minRadius.
	rpsOkay := false
	minDeltaBT := 1e12
	minDeltaBR := 1e12
	minDeltaRp := 1e12
	var minBT1, minBT2, minBR1, minBR2, minAssocψ, minRp1, minRp2 float64
	var minega1Vin, minega1Vout, minega2Vin, minega2Vout float64
	// The following store the information for when the maximum radius of periapsis is reached for both.
	var maxBT1, maxBT2, maxBR1, maxBR2, maxAssocψ, maxRp1, maxRp2 float64
	var maxega1Vin, maxega1Vout, maxega2Vin, maxega2Vout float64
	for ψ := step; ψ < 2*math.Pi; ψ += step {
		sψ, cψ := math.Sincos(ψ)
		vInfOutEGA1VNC := []float64{vInfInEGA1Norm * math.Cos(math.Pi-theta), vInfInEGA1Norm * math.Sin(math.Pi-theta) * cψ, -vInfInEGA1Norm * math.Sin(math.Pi-theta) * sψ}
		vInfOutEGA1Eclip := MxV33(transposedDCM.T(), vInfOutEGA1VNC)
		_, rP1, bT1, bR1, _, _ := smd.GAFromVinf(vInfInEGA1, vInfOutEGA1Eclip, smd.Earth)

		vInfInEGA2Eclip := make([]float64, 3)
		for i := 0; i < 3; i++ {
			vInfInEGA2Eclip[i] = vInfOutEGA1Eclip[i] + earthAtEGA1.V()[i] - earthAtEGA2.V()[i]
		}
		_, rP2, bT2, bR2, _, _ := smd.GAFromVinf(vInfInEGA2Eclip, vInfOutEGA2, smd.Earth)
		data += fmt.Sprintf("%f\t%f\t%f\n", ψ*r2d, rP1, rP2)
		if !rpsOkay && rP1 > minRadius && rP2 > minRadius {
			rpsOkay = true
			fmt.Printf("[OK ] ψ=%.6f\trP1=%.3f km\trP2=%.3f km\n", ψ*r2d, rP1, rP2)
		}
		if rpsOkay {
			// Compute the delta BT and BR so we can choose the smallest one.
			if math.Abs(bT1-bT2) < minDeltaBT && math.Abs(bR1-bR2) < minDeltaBR {
				//fmt.Printf("[NEW] deltaBt = %f => %v\tdeltaBr = %f => %v\n", math.Abs(bT1-bT2), math.Abs(bT1-bT2) < minDeltaBT, math.Abs(bR1-bR2), math.Abs(bR1-bR2) < minDeltaBR)
				// New mins!
				minDeltaBT = math.Abs(bT1 - bT2)
				minDeltaBR = math.Abs(bR1 - bR2)
				// Store them
				minBT1 = bT1
				minBR1 = bR1
				minRp1 = rP1
				minega1Vin = norm(vInfInEGA1)
				minega1Vout = norm(vInfOutEGA1Eclip)
				minBT2 = bT2
				minBR2 = bR2
				minRp2 = rP2
				minega2Vin = norm(vInfInEGA2Eclip)
				minega2Vout = norm(vInfOutEGA2)
				minAssocψ = ψ
			} else {
				//fmt.Printf("deltaBt = %f => %v\tdeltaBr = %f => %v\n", math.Abs(bT1-bT2), math.Abs(bT1-bT2) < minDeltaBT, math.Abs(bR1-bR2), math.Abs(bR1-bR2) < minDeltaBR)
			}
			if math.Abs(rP1-rP2) < minDeltaRp {
				// Just reached a new high for both rPs.
				minDeltaRp = math.Abs(rP1 - rP2)
				// Store them
				maxBT1 = bT1
				maxBR1 = bR1
				maxRp1 = rP1
				maxega1Vin = norm(vInfInEGA1)
				maxega1Vout = norm(vInfOutEGA1Eclip)
				maxBT2 = bT2
				maxBR2 = bR2
				maxRp2 = rP2
				maxega2Vin = norm(vInfInEGA2Eclip)
				maxega2Vout = norm(vInfOutEGA2)
				maxAssocψ = ψ
			}
			if rP1 < minRadius || rP2 < minRadius {
				rpsOkay = false
				fmt.Printf("[NOK] ψ=%.6f\trP1=%.3f km\trP2=%.3f km\n", ψ*r2d, rP1, rP2)
			}
		}
	}
	fmt.Printf("=== Mins: ψ=%f ===\nEGA1: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n\nEGA2: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n", minAssocψ*r2d, minBT1, minBR1, minRp1, minega1Vin, minega1Vout, minega1Vout-minega1Vin, minBT2, minBR2, minRp2, minega2Vin, minega2Vout, minega2Vout-minega2Vin)

	fmt.Printf("=== Max: ψ=%f ===\nEGA1: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n\nEGA2: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n", maxAssocψ*r2d, maxBT1, maxBR1, maxRp1, maxega1Vin, maxega1Vout, maxega1Vout-maxega1Vin, maxBT2, maxBR2, maxRp2, maxega2Vin, maxega2Vout, maxega2Vout-maxega2Vin)

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
func MxV33(m mat64.Matrix, v []float64) (o []float64) {
	vVec := mat64.NewVector(len(v), v)
	var rVec mat64.Vector
	rVec.MulVec(m, vVec)
	return []float64{rVec.At(0, 0), rVec.At(1, 0), rVec.At(2, 0)}
}
