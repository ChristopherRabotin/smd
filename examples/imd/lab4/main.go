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
	launchWindow := 500  //days
	arrivalWindow := 500 //days
	/*** END CONFIG ***/

	if launchWindow != arrivalWindow {
		panic("launch and arrival window must be of same length for now")
	}
	// Stores the content of the dat file.
	dat := fmt.Sprintf("%% %s -> %s\n%%arrival departure c3 vInf tof\n", initPlanet, arrivalPlanet)

	for launchDay := 0; launchDay < launchWindow; launchDay++ {
		launchDT := initLaunch.Add(time.Duration(launchDay*24) * time.Hour)
		fmt.Printf("Launch date %s\n", launchDT)
		initOrbit := initPlanet.HelioOrbit(launchDT)
		initR := mat64.NewVector(3, initOrbit.R())
		initV := mat64.NewVector(3, initOrbit.V())
		for arrivalDay := 0; arrivalDay < arrivalWindow; arrivalDay++ {
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
			dat += fmt.Sprintf("%d %d %f %f %f\n", launchDay, arrivalDay, c3, vInfArrival, tof.Hours()/24)
		}
	}

	// Write CSV file.
	f, err := os.Create("./contour.dat")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(dat); err != nil {
		panic(err)
	}

}
