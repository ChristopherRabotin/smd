package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

func main() {
	launchDT := time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC)
	// Let's set an initial estimated arrival time guess.
	arrivalEstDT := launchDT.Add(time.Duration(100*24) * time.Hour)
	// RV() is a pointer method (because of the cache update)
	earth := smd.Earth.HelioOrbit(launchDT)
	Rearth := mat64.NewVector(3, earth.R())
	Vearth := mat64.NewVector(3, earth.V())
	for _, ttype := range []smd.TransferType{smd.TType1, smd.TType2} {
		// Initialize the CSV string
		csvContent := fmt.Sprintf("# Earth -> Mars Lambert type %s\n#Launch: %s\n#Initial arrival:%s\ndays,c3,vInf,phi2\n", ttype, launchDT, arrivalEstDT)
		minC3 := 10e4
		minVinf := 10e4
		minDurC3 := 0.
		minDurVinf := 0.
		for days := 0; days < 200; days++ {
			arrivalDT := arrivalEstDT.Add(time.Duration(days) * 24 * time.Hour)
			duration := arrivalDT.Sub(launchDT)
			Rmars := mat64.NewVector(3, smd.Mars.HelioOrbit(arrivalDT).R())
			Vi, _, ψ, err := smd.Lambert(Rearth, Rmars, duration, ttype, smd.Sun)
			if err != nil {
				fmt.Printf("[ERROR] %s: %s\n", duration, err)
				continue
			}
			// Compute the v_infinity
			vInfVec := mat64.NewVector(3, nil)
			vInfVec.SubVec(Vi, Vearth)
			vInf := mat64.Norm(vInfVec, 2)
			c3 := math.Pow(vInf, 2)
			// Add to CSV
			tof := arrivalDT.Sub(launchDT).Hours() / 24
			csvContent += fmt.Sprintf("%f,%f,%f,%f\n", tof, c3, vInf, math.Pow(ψ, 2))
			// Check if min
			if vInf < minVinf {
				minVinf = vInf
				minDurVinf = tof
			}
			if c3 < minC3 {
				minC3 = c3
				minDurC3 = tof
			}
		}
		fmt.Printf("===== %s min ======\nvInf=%f\tdur=%.0f\nc3=%f\tdur=%.0f\n=======================\n", ttype, minVinf, minDurVinf, minC3, minDurC3)
		// Write CSV file.
		f, err := os.Create(fmt.Sprintf("./pb1-%s.csv", ttype))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(csvContent); err != nil {
			panic(err)
		}
	}
}
