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
	pcpName := "lab4pcp2016"
	initPlanet := smd.Earth
	arrivalPlanet := smd.Mars
	initLaunch := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	initArrival := time.Date(2016, 04, 30, 0, 0, 0, 0, time.UTC)
	maxLaunch := time.Date(2016, 6, 30, 0, 0, 0, 0, time.UTC)
	maxArrival := time.Date(2017, 2, 5, 0, 0, 0, 0, time.UTC)
	launchWindow := int(maxLaunch.Sub(initLaunch).Hours() / 24)    //days
	arrivalWindow := int(maxArrival.Sub(initArrival).Hours() / 24) //days
	/*** END CONFIG ***/

	// Stores the content of the dat file.
	// No trailing new line because it's add in the for loop.
	dat := fmt.Sprintf("%% %s -> %s\n%%arrival days as new lines, departure as new columns", initPlanet, arrivalPlanet)
	hdls := make([]*os.File, 4)
	for i, name := range []string{"c3", "tof", "vinf", "dates"} {
		// Write CSV file.
		f, err := os.Create(fmt.Sprintf("./contour-%s-%s.dat", pcpName, name))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(dat); err != nil {
			panic(err)
		}
		hdls[i] = f
	}

	// Let's write the date information now and close that file.
	hdls[3].WriteString(fmt.Sprintf("\n%%departure: \"%s\"\n%%arrival: \"%s\"\n%d,%d\n%d,%d\n", initLaunch.Format("2006-Jan-02"), initArrival.Format("2006-Jan-02"), 1, launchWindow, 1, arrivalWindow))
	hdls[3].Close()

	for launchDay := 0; launchDay < launchWindow; launchDay++ {
		// New line in files
		for _, hdl := range hdls[:3] {
			if _, err := hdl.WriteString("\n"); err != nil {
				panic(err)
			}
		}

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
			Vi, Vf, _, err := smd.Lambert(initR, arrivalR, tof, smd.TTypeAuto, smd.Sun)
			var c3, vInfArrival float64
			if err != nil {
				fmt.Printf("departure: %s\tarrival: %s\t\t%s\n", launchDT, arrivalDT, err)
				c3 = math.NaN()
				vInfArrival = math.NaN()
			} else {
				// Compute the c3
				VInfInit := mat64.NewVector(3, nil)
				VInfInit.SubVec(Vi, initV)
				c3 = math.Pow(mat64.Norm(VInfInit, 2), 2)
				if math.IsNaN(c3) {
					c3 = 0
				}
				// Compute the v_infinity at destination
				VInfArrival := mat64.NewVector(3, arrivalOrbit.V())
				VInfArrival.SubVec(Vf, VInfArrival)
				vInfArrival = mat64.Norm(VInfArrival, 2)
			}
			// Store data
			hdls[0].WriteString(fmt.Sprintf("%f,", c3))
			hdls[1].WriteString(fmt.Sprintf("%f,", tof.Hours()/24))
			hdls[2].WriteString(fmt.Sprintf("%f,", vInfArrival))
		}
	}
	// Print the matlab command to help out
	fmt.Printf("=== MatLab ===\npcpplots('%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"))
}
