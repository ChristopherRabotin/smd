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
	/*initLaunch := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	initArrival := time.Date(2016, 04, 30, 0, 0, 0, 0, time.UTC)
	maxLaunch := time.Date(2016, 6, 30, 0, 0, 0, 0, time.UTC)
	maxArrival := time.Date(2017, 2, 5, 0, 0, 0, 0, time.UTC)*/

	// PCP #1 of lab 6
	initPlanet := smd.Earth
	arrivalPlanet := smd.Jupiter
	//initLaunch := julian.JDToTime(2453714.5)
	initLaunch := time.Date(2006, 1, 6, 0, 0, 0, 0, time.UTC)
	initArrival := julian.JDToTime(2454129.5)
	//maxLaunch := julian.JDToTime(2453794.5)
	maxLaunch := time.Date(2006, 1, 7, 0, 0, 0, 0, time.UTC)
	maxArrival := julian.JDToTime(2454239.5)
	c3MapPCP1, tofMapPCP1, vinfMapPCP1 := pcpGenerator(initPlanet, arrivalPlanet, initLaunch, maxLaunch, initArrival, maxArrival, 24*4, 1, true, "lab6pcp1", true)

	//PCP #2 of lab 6
	initPlanet = smd.Jupiter
	arrivalPlanet = smd.Pluto
	initLaunchJup := julian.JDToTime(2454129.5)
	initArrival = julian.JDToTime(2456917.5)
	maxLaunchJup := julian.JDToTime(2454239.5)
	maxArrival = julian.JDToTime(2457517.5)

	/*** END CONFIG ***/
	/*vinfInitMapPCP2, tofMapPCP2, vinfArrMapPCP2 :=*/
	pcpGenerator(initPlanet, arrivalPlanet, initLaunchJup, maxLaunchJup, initArrival, maxArrival, 1, 1, false, "lab6pcp2", true)

	// Lab6 searching
	earthDepart := time.Date(2006, 1, 9, 0, 0, 0, 0, time.UTC)
	minC3 := 1e6 // Arbitrary large value
	maxC3 := 180.
	minDT := time.Now()
	for launchDT, c3PerDay := range c3MapPCP1 {
		for arrivalIdx, c3 := range c3PerDay {
			if c3 > maxC3 {
				continue // Cannot use this data point
			}
			arrivalTOF := tofMapPCP1[launchDT][arrivalIdx]
			arrivalDT := launchDT.Add(time.Duration(arrivalTOF*24) * time.Hour)
			if arrivalDT.After(maxArrival) {
				continue
			}
			// All constraints are met
			if c3 < minC3 {
				minDT = launchDT
				minC3 = c3
			}
		}
	}
	fmt.Printf("=== MIN ===\nDT: %s\tc3=%.3f km^2/s^2\n", minDT, minC3)
	_, c3ok := c3MapPCP1[earthDepart]
	_, vinfok := vinfMapPCP1[earthDepart]
	_, tofok := tofMapPCP1[earthDepart]
	fmt.Printf("PCP1 existance: %v %v %v\n", c3ok, vinfok, tofok)
}

func pcpGenerator(initPlanet, arrivalPlanet smd.CelestialObject, initLaunch, maxLaunch, initArrival, maxArrival time.Time, ptsPerLaunchDay, ptsPerArrivalDay float64, plotC3 bool, pcpName string, verbose bool) (c3Map, tofMap, vinfMap map[time.Time][]float64) {
	launchWindow := int(maxLaunch.Sub(initLaunch).Hours() / 24)    //days
	arrivalWindow := int(maxArrival.Sub(initArrival).Hours() / 24) //days
	// Create the output arrays
	c3Map = make(map[time.Time][]float64)
	tofMap = make(map[time.Time][]float64)
	vinfMap = make(map[time.Time][]float64)
	if verbose {
		fmt.Printf("Launch window: %d days\nArrival window: %d days\n", launchWindow, arrivalWindow)
	}
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

	for launchDay := 0.; launchDay < float64(launchWindow); launchDay += 1 / ptsPerLaunchDay {
		// New line in files
		for _, hdl := range hdls[:3] {
			if _, err := hdl.WriteString("\n"); err != nil {
				panic(err)
			}
		}
		launchDT := initLaunch.Add(time.Duration(launchDay*24*3600) * time.Second)
		if verbose {
			fmt.Printf("Launch date %s\n", launchDT)
		}
		// Initialize the values
		c3Map[launchDT] = make([]float64, arrivalWindow)
		tofMap[launchDT] = make([]float64, arrivalWindow)
		vinfMap[launchDT] = make([]float64, arrivalWindow)

		initOrbit := initPlanet.HelioOrbit(launchDT)
		initPlanetR := mat64.NewVector(3, initOrbit.R())
		initPlanetV := mat64.NewVector(3, initOrbit.V())
		arrivalIdx := 0
		for arrivalDay := 0.; arrivalDay < float64(arrivalWindow); arrivalDay += 1 / ptsPerArrivalDay {
			arrivalDT := initArrival.Add(time.Duration(arrivalDay*24) * time.Hour)
			arrivalOrbit := arrivalPlanet.HelioOrbit(arrivalDT)
			arrivalR := mat64.NewVector(3, arrivalOrbit.R())
			arrivalV := mat64.NewVector(3, arrivalOrbit.V())

			tof := arrivalDT.Sub(launchDT)
			Vi, Vf, _, err := smd.Lambert(initPlanetR, arrivalR, tof, smd.TTypeAuto, smd.Sun)
			var c3, vInfArrival float64
			if err != nil {
				if verbose {
					fmt.Printf("departure: %s\tarrival: %s\t\t%s\n", launchDT, arrivalDT, err)
				}
				c3 = math.NaN()
				vInfArrival = math.NaN()
			} else {
				// Compute the c3
				VInfInit := mat64.NewVector(3, nil)
				VInfInit.SubVec(initPlanetV, Vi)
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
				VInfArrival := mat64.NewVector(3, nil)
				VInfArrival.SubVec(Vf, arrivalV)
				vInfArrival = mat64.Norm(VInfArrival, 2)
			}
			// Store data in the files
			hdls[0].WriteString(fmt.Sprintf("%f,", c3))
			hdls[1].WriteString(fmt.Sprintf("%f,", tof.Hours()/24))
			hdls[2].WriteString(fmt.Sprintf("%f,", vInfArrival))
			// and in the arrays
			c3Map[launchDT][arrivalIdx] = c3
			tofMap[launchDT][arrivalIdx] = tof.Hours() / 24
			vinfMap[launchDT][arrivalIdx] = vInfArrival
			arrivalIdx++
		}
	}
	if verbose {
		// Print the matlab command to help out
		if plotC3 {
			fmt.Printf("=== MatLab ===\npcpplots('%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), arrivalPlanet.Name)
		} else {
			fmt.Printf("=== MatLab ===\npcpplotsVinfs('%s', '%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), initPlanet.Name, arrivalPlanet.Name)
		}
	}
	return
}
