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
	/***** CONFIG ******/
	launchDT := time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC)
	departurePlanet := smd.Earth
	arrivalEstDT := launchDT.Add(time.Duration(100*24) * time.Hour)
	arrivalPlanet := smd.Mars
	exportResults := false
	/*** END CONFIG ****/
	// RV() is a pointer method (because of the cache update)
	departureOrbit := departurePlanet.HelioOrbit(launchDT)
	Rdepart := mat64.NewVector(3, departureOrbit.R())
	Vdepart := mat64.NewVector(3, departureOrbit.V())
	for _, ttype := range []smd.TransferType{smd.TType1, smd.TType2} {
		// Initialize the CSV string
		csvContent := fmt.Sprintf("# %s -> %s Lambert type %s\n#Launch: %s\n#Initial arrival:%s\ndays,c3,vInf,phi2\n", departurePlanet, arrivalPlanet, ttype, launchDT, arrivalEstDT)
		minC3 := 10e4
		minVinf := 10e4
		var minArrivalDT time.Time
		for days := 0; days < 200; days++ {
			arrivalDT := arrivalEstDT.Add(time.Duration(days) * 24 * time.Hour)
			duration := arrivalDT.Sub(launchDT)
			Rmars := mat64.NewVector(3, arrivalPlanet.HelioOrbit(arrivalDT).R())
			Vi, _, ψ, err := smd.Lambert(Rdepart, Rmars, duration, ttype, smd.Sun)
			if err != nil {
				fmt.Printf("[ERROR] %s: %s\n", duration, err)
				continue
			}
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
