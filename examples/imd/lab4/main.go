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
	/*** CONFIG ***/
	initPlanet := smd.Earth
	initLaunch := time.Date(2005, 6, 1, 0, 0, 0, 0, time.UTC)
	arrivalPlanet := smd.Mars
	initArrival := time.Date(2005, 12, 1, 0, 0, 0, 0, time.UTC)
	window := 500 //days
	/*** END CONFIG ***/

	csvContent := fmt.Sprintf("# %s -> %s\nLaunchDay,ArrivalDay,tof,c3@%s,vInf@%s\n", initPlanet, arrivalPlanet, initPlanet.Name, arrivalPlanet.Name)
	for launchDay := 0; launchDay < window; launchDay++ {
		launchDT := initLaunch.Add(time.Duration(launchDay*24) * time.Hour)
		fmt.Printf("Launch date %s\n", launchDT)
		initOrbit := initPlanet.HelioOrbit(launchDT)
		initR := mat64.NewVector(3, initOrbit.R())
		initV := mat64.NewVector(3, initOrbit.V())
		for arrivalDay := 0; arrivalDay < window; arrivalDay++ {
			arrivalDT := initArrival.Add(time.Duration(arrivalDay*24) * time.Hour)
			arrivalOrbit := arrivalPlanet.HelioOrbit(arrivalDT)
			arrivalR := mat64.NewVector(3, arrivalOrbit.R())

			tof := arrivalDT.Sub(launchDT)
			Vi, Vf, _, err := smd.Lambert(initR, arrivalR, tof, smd.TType2, smd.Sun)
			if err != nil {
				fmt.Println(err)
				break
			}
			// Compute the c3
			VInfInit := mat64.NewVector(3, nil)
			VInfInit.SubVec(Vi, initV)
			c3 := math.Pow(mat64.Norm(VInfInit, 2), 2)
			if math.IsNaN(c3) {
				c3 = 0
			}
			// Compute the v_infinity at destination
			VInfArrival := mat64.NewVector(3, nil)
			VInfArrival.SubVec(VInfArrival, Vf)
			vInfArrival := mat64.Norm(VInfArrival, 2)
			// Add to CSV
			csvContent += fmt.Sprintf("%d,%d,%f,%f,%f\n", launchDay, arrivalDay, tof.Hours()/24, c3, vInfArrival)
		}
	}

	// Write CSV file.
	f, err := os.Create("./contourdata.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(csvContent); err != nil {
		panic(err)
	}
}
