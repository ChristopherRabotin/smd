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
	/*** CONFIG ***/
	plotC3 := false // Set to false to plot the two vInfs at initial and arrival planets
	pcpName := "lab6pcp2"
	initPlanet := smd.Jupiter
	arrivalPlanet := smd.Pluto
	/*initLaunch := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	initArrival := time.Date(2016, 04, 30, 0, 0, 0, 0, time.UTC)
	maxLaunch := time.Date(2016, 6, 30, 0, 0, 0, 0, time.UTC)
	maxArrival := time.Date(2017, 2, 5, 0, 0, 0, 0, time.UTC)*/

	/* // PCP #1 of lab 6
	initLaunch := julian.JDToTime(2453714.5)
	initArrival := julian.JDToTime(2454129.5)
	maxLaunch := julian.JDToTime(2453794.5)
	maxArrival := julian.JDToTime(2454239.5)
	*/
	//PCP #2 of lab 6
	initLaunch := julian.JDToTime(2454129.5)
	initArrival := julian.JDToTime(2456917.5)
	maxLaunch := julian.JDToTime(2454239.5)
	maxArrival := julian.JDToTime(2457517.5)

	/*** END CONFIG ***/

	launchWindow := int(maxLaunch.Sub(initLaunch).Hours() / 24)    //days
	arrivalWindow := int(maxArrival.Sub(initArrival).Hours() / 24) //days
	fmt.Printf("Launch window: %d days\nArrival window: %d days\n", launchWindow, arrivalWindow)

	// Stores the content of the dat file.
	// No trailing new line because it's add in the for loop.
	dat := fmt.Sprintf("%% %s -> %s\n%%arrival days as new lines, departure as new columns", initPlanet, arrivalPlanet)
	hdls := make([]*os.File, 4)
	var fNames []string
	if plotC3 {
		fNames = []string{"c3", "tof", "vinf", "dates"}
	} else {
		fNames = []string{"vinf-init", "tof", "vinf-arrival", "dates"}
	}
	for i, name := range fNames {
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
				// WARNING: When *not* plotting the c3, we just store the V infinity at departure in the c3 variable!
				if plotC3 {
					c3 = math.Pow(mat64.Norm(VInfInit, 2), 2)
				} else {
					c3 = mat64.Norm(VInfInit, 2)
				}
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
	if plotC3 {
		fmt.Printf("=== MatLab ===\npcpplots('%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), arrivalPlanet.Name)
	} else {
		fmt.Printf("=== MatLab ===\npcpplotsVinfs('%s', '%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), initPlanet.Name, arrivalPlanet.Name)
	}
}
