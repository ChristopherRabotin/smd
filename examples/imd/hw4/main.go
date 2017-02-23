package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

const (
	questionNumber = 3
	r2d            = 180 / math.Pi
	d2r            = 1 / r2d
)

func main() {
	if questionNumber == 1 {
		// Question 1
		rVenus := []float64{-96948447.3751, 46106976.1901, 0}
		vVenus := []float64{-15.1945, -31.7927, 0}
		vSC := []float64{-10.8559, -35.9372, 0}
		// a
		flyby := smd.NewOrbitFromRV(rVenus, vSC, smd.Sun)
		ξpre := flyby.Energyξ()
		fmt.Printf("ξ = %f\n", ξpre)
		// b
		vInfVec := make([]float64, 3)
		for i := 0; i < 3; i++ {
			vInfVec[i] = vSC[i] - vVenus[i]
		}
		vInf := norm(vInfVec)
		f1b := initCSV("turnangle1b")
		f1b.WriteString("rVenus (km), rP (km), turning angle (degrees), energy (pre), energy (trailing), energy (leading)\n")
		for rP := 0.; rP < 200000; rP++ {
			ψ := smd.GATurnAngle(vInf, rP, smd.Venus) // in radians
			// Build vectors
			vInfVecLeading := smd.MxV33(smd.R3(-ψ), vInfVec)
			vInfVecTrailing := smd.MxV33(smd.R3(ψ), vInfVec)
			vVecLeading := make([]float64, 3)
			vVecTrailing := make([]float64, 3)
			for i := 0; i < 3; i++ {
				vVecLeading[i] = vInfVecLeading[i] + vVenus[i]
				vVecTrailing[i] = vInfVecTrailing[i] + vVenus[i]
			}
			ξleading := smd.NewOrbitFromRV(rVenus, vVecLeading, smd.Sun).Energyξ()
			ξtrailing := smd.NewOrbitFromRV(rVenus, vVecTrailing, smd.Sun).Energyξ()
			f1b.WriteString(fmt.Sprintf("%.0f,%.0f,%.3f,%.6f,%.6f,%.6f,\n", smd.Venus.Radius, rP, ψ*r2d, ξpre, ξtrailing, ξleading))
		}
		f1b.Close()
	} else if questionNumber == 2 {
		vInfInVec := []float64{-5.19425, 5.19424, -5.19425}
		vInfOutVec := []float64{-8.58481, 1.17067, -2.42304}
		ψ, rP, bT, bR, B, θ := smd.GAFromVinf(vInfInVec, vInfOutVec, smd.Earth)
		fmt.Printf("ψ=%.3f deg\trP=%.3f km\tbT=%.3f km\tbR=%.3f km\tB=%.3f km\tθ=%.3f deg\n", ψ*r2d, rP, bT, bR, B, θ*r2d)
	} else {
		dtLaunch := time.Date(1989, 10, 8, 0, 0, 0, 0, time.UTC)
		dtVenusFB := time.Date(1990, 2, 10, 0, 0, 0, 0, time.UTC)
		dtEarthFB1 := time.Date(1990, 12, 10, 0, 0, 0, 0, time.UTC)
		/*dtEarthFB2 := time.Date(1992, 12, 9, 0, 0, 0, 0, time.UTC)
		dtJupiter := time.Date(1996, 3, 21, 0, 0, 0, 0, time.UTC)*/

		launchR := mat64.NewVector(3, smd.Earth.HelioOrbit(dtLaunch).R()) // Position of Earth at Launch
		venusRf, venusVf := smd.Venus.HelioOrbit(dtVenusFB).RV()          // Position of Venus at flyby
		venusR := mat64.NewVector(3, venusRf)
		venusV := mat64.NewVector(3, venusVf)
		earthRFB1 := mat64.NewVector(3, smd.Earth.HelioOrbit(dtEarthFB1).R()) // Position of Earth at first flyby

		_, VfEarth2Venus, _, err := smd.Lambert(launchR, venusR, dtVenusFB.Sub(dtLaunch), smd.TTypeAuto, smd.Sun)
		if err != nil {
			panic(err)
		}
		ViVenus2Earth, _, _, err := smd.Lambert(venusR, earthRFB1, dtEarthFB1.Sub(dtVenusFB), smd.TTypeAuto, smd.Sun)
		if err != nil {
			panic(err)
		}
		// Compute the v infinities
		var vInfInVec, vInfOutVec mat64.Vector
		vInfInVec.SubVec(venusV, VfEarth2Venus)
		vInfOutVec.SubVec(venusV, ViVenus2Earth)
		// Extract the data
		vInfIn := make([]float64, 3)
		vInfOut := make([]float64, 3)
		for i := 0; i < 3; i++ {
			vInfIn[i] = vInfInVec.At(i, 0)
			vInfOut[i] = vInfOutVec.At(i, 0)
		}
		ψ, rP, bT, bR, B, θ := smd.GAFromVinf(vInfIn, vInfOut, smd.Venus)
		fmt.Printf("ψ=%.3f deg\trP=%.3f km\tbT=%.3f km\tbR=%.3f km\tB=%.3f km\tθ=%.3f deg\n", ψ*r2d, rP, bT, bR, B, θ*r2d)
	}
}

func initCSV(fname string) *os.File {
	f, err := os.Create(fmt.Sprintf("./%s.csv", fname))
	if err != nil {
		panic(err)
	}
	return f
}

// norm returns the norm of a given vector which is supposed to be 3x1.
func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}
