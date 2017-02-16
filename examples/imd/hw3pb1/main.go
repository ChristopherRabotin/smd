package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/soniakeys/meeus/julian"
)

func main() {
	jde := 2454085.5
	launchDT := julian.JDToTime(jde)
	arrivalDT := launchDT.Add(time.Duration(830*24) * time.Hour)
	marsOrbit := smd.Mars.HelioOrbit(launchDT)
	_, marsV := marsOrbit.RV()
	Vdepart := mat64.NewVector(3, marsV)
	//jupiterOrbit := smd.Jupiter.HelioOrbit(arrivalDT)
	//_, _ := jupiterOrbit.RV()
	fmt.Printf("==== Mars @%s ====\n", julian.JDToTime(jde))
	fmt.Printf("==== Jupiter @%s ====\n", arrivalDT)
	departurePlanet := smd.Mars
	arrivalEstDT := launchDT.Add(time.Duration(200*24) * time.Hour)
	arrivalPlanet := smd.Jupiter
	exportResults := true
	window := 4000 // in days
	step := time.Duration(24) * time.Hour
	/*** END CONFIG ****/

	// DEBUG
	marsR := mat64.NewVector(3, []float64{-1.2817e8, -1.9059e8, -0.0084e8})
	jupiterR := mat64.NewVector(3, []float64{4.8338e8, -5.8746e8, -0.0838e8})
	// END DEBUG

	// The following is mostly copied from hw2pb1
	fmt.Printf("==== Lambert min solver ====\n%s -> %s\nLaunch:%s \tWindow: %d days\n\n", departurePlanet, arrivalPlanet, launchDT, window)
	// Initialize the CSV string
	csvContent := fmt.Sprintf("# %s -> %s\ndays,c3,vInf,phi2\n", departurePlanet, arrivalPlanet)

	for _, ttype := range []smd.TransferType{smd.TType1, smd.TType2, smd.TType3, smd.TType4} {
		minC3 := 10e4
		minVinf := 10e4
		var minArrivalDT time.Time
		arrivalDT := arrivalEstDT
		maxDT := arrivalEstDT.Add(time.Duration(window) * 24 * time.Hour)
		for ; arrivalDT.Before(maxDT); arrivalDT = arrivalDT.Add(step) {
			duration := arrivalDT.Sub(launchDT)
			//Rmars := mat64.NewVector(3, arrivalPlanet.HelioOrbit(arrivalDT).R())

			Vi, _, ψ, _ := smd.Lambert(marsR, jupiterR, duration, ttype, smd.Sun)
			// Compute the v_infinity
			vInfVec := mat64.NewVector(3, nil)
			vInfVec.SubVec(Vi, Vdepart)
			vInf := mat64.Norm(vInfVec, 2)
			c3 := math.Pow(vInf, 2)
			if math.IsNaN(vInf) {
				vInf = 0
				c3 = 0
			}
			// Add to CSV
			tof := arrivalDT.Sub(launchDT).Hours() / 24
			if exportResults {
				csvContent += fmt.Sprintf("%f,%f,%f,%f\n", tof, c3, vInf, ψ)
			}
			// Check if min
			if vInf != 0 && vInf < minVinf {
				minVinf = vInf
				minC3 = c3
				minArrivalDT = arrivalDT
			}
		}
		fmt.Printf("==== Minimum for %s ====\nArrival=%s (%.0f days)\nvInf=%f km/s\tc3=%f km^2/s^2\n\n", ttype, minArrivalDT, minArrivalDT.Sub(launchDT).Hours()/24, minVinf, minC3)
	}
	if exportResults {
		// Write CSV file.
		//f, err := os.Create(fmt.Sprintf("./pb1-%s.csv", ttype))
		f, err := os.Create("./pb1.csv")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(csvContent); err != nil {
			panic(err)
		}
	}

}
