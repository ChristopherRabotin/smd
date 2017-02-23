package main

import (
	"fmt"
	"math"
	"os"

	"github.com/ChristopherRabotin/smd"
)

const (
	questionNumber = 2
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
			ψ := smd.GATurnAngle(vInf, rP, smd.Venus) * r2d // in degrees
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
			f1b.WriteString(fmt.Sprintf("%.0f,%.0f,%.3f,%.6f,%.6f,%.6f,\n", smd.Venus.Radius, rP, ψ, ξpre, ξtrailing, ξleading))
		}
		f1b.Close()
	} else if questionNumber == 2 {

	} else {
		fmt.Printf("TODO")
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
