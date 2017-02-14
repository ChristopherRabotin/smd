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
	marsR, marsV := marsOrbit.RV()
	jupiterOrbit := smd.Jupiter.HelioOrbit(arrivalDT)
	jupiterR, jupiterV := jupiterOrbit.RV()
	fmt.Printf("==== Mars @%s ====\nR=%+v km\tV=%+v km/s\n", julian.JDToTime(jde), marsR, marsV)
	fmt.Printf("==== Jupiter @%s ====\nR=%+v km\tV=%+v km/s\n", arrivalDT, jupiterR, jupiterV)
	departurePlanet := smd.Mars
	arrivalEstDT := launchDT.Add(time.Duration(100*24) * time.Hour)
	arrivalPlanet := smd.Jupiter
	exportResults := false
	window := 1500 // in days
	step := time.Duration(6) * time.Hour
	/*** END CONFIG ****/
	// The following is mostly copied from hw2pb1
	fmt.Printf("==== Lambert min solver ====\n%s -> %s\nLaunch:%s \tWindow: %d days\n\n", departurePlanet, arrivalPlanet, launchDT, window)
	for _, ttype := range []smd.TransferType{smd.TType1, smd.TType2, smd.TType3, smd.TType4} {
		rDep, vDep := departurePlanet.HelioOrbit(launchDT).RV()
		if ttype.Revs() > 0 {
			// Let's find the position and velocity the planet will have a bit later.
			p := arrivalPlanet.HelioOrbit(launchDT).Period()
			arrivalEstDT = arrivalEstDT.Add(p / 2)
		}
		Rdepart := mat64.NewVector(3, rDep)
		Vdepart := mat64.NewVector(3, vDep)
		// Initialize the CSV string
		csvContent := fmt.Sprintf("# %s -> %s Lambert type %s\n#Launch: %s\n#Initial arrival:%s\ndays,c3,vInf,phi2\n", departurePlanet, arrivalPlanet, ttype, launchDT, arrivalEstDT)
		minC3 := 10e4
		minVinf := 10e4
		var minArrivalDT time.Time
		arrivalDT := arrivalEstDT
		maxDT := arrivalEstDT.Add(time.Duration(window) * 24 * time.Hour)
		for ; arrivalDT.Before(maxDT); arrivalDT = arrivalDT.Add(step) {
			duration := arrivalDT.Sub(launchDT)
			Rmars := mat64.NewVector(3, arrivalPlanet.HelioOrbit(arrivalDT).R())
			Vi, _, ψ, _ := smd.Lambert(Rdepart, Rmars, duration, ttype, smd.Sun)
			// Compute the v_infinity
			vInfVec := mat64.NewVector(3, nil)
			vInfVec.SubVec(Vi, Vdepart)
			vInf := mat64.Norm(vInfVec, 2)
			c3 := math.Pow(vInf, 2)
			// Add to CSV
			tof := arrivalDT.Sub(launchDT).Hours() / 24
			if exportResults {
				csvContent += fmt.Sprintf("%f,%f,%f,%f\n", tof, c3, vInf, math.Pow(ψ, 2))
			}
			// Check if min
			if vInf < minVinf {
				minVinf = vInf
				minC3 = c3
				minArrivalDT = arrivalDT
			}
		}
		fmt.Printf("==== Minimum for %s ====\nArrival=%s (%.0f days)\nvInf=%f km/s\tc3=%f km^2/s^2\n\n", ttype, minArrivalDT, minArrivalDT.Sub(launchDT).Hours()/24, minVinf, minC3)
		if exportResults {
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
}
