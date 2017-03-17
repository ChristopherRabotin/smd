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

const (
	finder = true
)

func main() {
	/*** CONFIG ***/
	/*
		initPlanet := smd.Earth
		arrivalPlanet := smd.Mars
		initLaunch := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
		initArrival := time.Date(2016, 04, 30, 0, 0, 0, 0, time.UTC)
		maxLaunch := time.Date(2016, 6, 30, 0, 0, 0, 0, time.UTC)
		maxArrival := time.Date(2017, 2, 5, 0, 0, 0, 0, time.UTC)
		pcpGenerator(initPlanet, arrivalPlanet, initLaunch, maxLaunch, initArrival, maxArrival, 1, 1, true, "lab4pcp0", true)
	*/
	// PCP #1 of lab 6
	initPlanet := smd.Earth
	arrivalPlanet := smd.Jupiter
	//initLaunch := julian.JDToTime(2453714.5)
	initLaunch := time.Date(2006, 1, 9, 0, 0, 0, 0, time.UTC)
	initArrival := julian.JDToTime(2454129.5)
	//maxLaunch := julian.JDToTime(2453794.5)
	maxLaunch := time.Date(2006, 1, 10, 0, 0, 0, 0, time.UTC)
	maxArrival := julian.JDToTime(2454239.5)

	resolutionJupiter := 4.
	c3MapPCP1, tofMapPCP1, vinfMapPCP1, _, vinfMapVecsPCP1 := pcpGenerator(initPlanet, arrivalPlanet, initLaunch, maxLaunch, initArrival, maxArrival, 24*1, resolutionJupiter, true, "lab6pcp1", false)

	//PCP #2 of lab 6
	/*initPlanet = smd.Jupiter
	arrivalPlanet = smd.Pluto*/
	//initLaunchJup := julian.JDToTime(2454129.5)
	initArrivalPluto := julian.JDToTime(2456917.5)
	//maxLaunchJup := julian.JDToTime(2454239.5)
	maxArrivalPluto := julian.JDToTime(2457517.5)
	//pcpGenerator(smd.Jupiter, smd.Pluto, initLaunchJup, maxLaunchJup, initArrivalPluto, maxArrivalPluto, 24*1, 24*1, false, "lab6pcp2", true)
	if !finder {
		return
	}
	/*** END CONFIG ***/

	// Lab6 searching
	//earthDepart := time.Date(2006, 1, 9, 0, 0, 0, 0, time.UTC)
	maxC3 := 180. // Fixed
	minC3 := 1e6  // Arbitrary large value
	minVinfInPluto := 1e6
	minEarthLaunchDT := time.Now()
	minPlutoArrivalDT := time.Now()
	/* The c3 is the most important because the lower it is, the more science payload mass we can have.
	So the forcing function here will simply be the sum of the c3 and the vInf at arrival. Some back of the enveloppe
	shows that should be good stuff.
	*/
	var mcVal, mcC3, mcVinfJGA, mcVinfPluto float64
	mcLaunch := time.Now()
	mcJGA := time.Now()
	mcPCE := time.Now()
	mcVal = 1e20 // large
	// With Rp at jupiter
	minRp := 30 * smd.Jupiter.Radius // Set to a negative value to not take that into consideration.
	maxRp := 0.0
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
			vinfInJupiter := vinfMapPCP1[launchDT][arrivalIdx]
			// All departure constraints seem to be met.
			fmt.Printf("start %s\tend %s\n", arrivalDT, arrivalDT.Add(24*time.Hour))
			vinfDepJMapPCP2, tofMapPCP2, vinfArrPMapPCP2, vinfMapVecsPCP2, _ := pcpGenerator(smd.Jupiter, smd.Pluto, arrivalDT, arrivalDT.Add(24*time.Hour), initArrivalPluto, maxArrivalPluto, resolutionJupiter, 1, false, "lab6pcp3tmp", false)
			// Go through solutions and move on with values which are within the constraints.
			for depJupDT, vInfDepPerDay := range vinfDepJMapPCP2 {
				for arrPlutIdx, vInfDepJup := range vInfDepPerDay {
					if math.Abs(vinfInJupiter-vInfDepJup) < 0.4 {
						if minRp > 0 {
							// Check if the rP is okay
							vInfIn := []float64{vinfMapVecsPCP1[launchDT][arrivalIdx].At(0, 0), vinfMapVecsPCP1[launchDT][arrivalIdx].At(1, 0), vinfMapVecsPCP1[launchDT][arrivalIdx].At(2, 0)}
							vInfOut := []float64{vinfMapVecsPCP2[depJupDT][arrPlutIdx].At(0, 0), vinfMapVecsPCP2[depJupDT][arrPlutIdx].At(1, 0), vinfMapVecsPCP2[depJupDT][arrPlutIdx].At(2, 0)}
							_, rp, _, _, _, _ := smd.GAFromVinf(vInfIn, vInfOut, smd.Jupiter)
							if rp > maxRp {
								maxRp = rp
							}
							if rp < minRp {
								//fmt.Printf("rP = %.2f * Jupiter radius (%.3f km)\t%+v\t%+v\n", rp/smd.Jupiter.Radius, rp, vInfIn, vInfOut)
								continue
							}
						}
						vinfArr := vinfArrPMapPCP2[depJupDT][arrPlutIdx]
						if vinfArr < 14.5 {
							if vinfArr < minVinfInPluto {
								minVinfInPluto = vinfArr
							}
							// Yay! All conditions work
							if c3 < minC3 {
								minEarthLaunchDT = launchDT
								minC3 = c3
							}
							//minPlutoArrivalDT
							plutoTOF := tofMapPCP2[depJupDT][arrivalIdx]
							plutoArrivalDT := arrivalDT.Add(time.Duration(plutoTOF*24) * time.Hour)
							if plutoArrivalDT.Before(minPlutoArrivalDT) {
								minPlutoArrivalDT = plutoArrivalDT
							}
							// Apply my forcing function
							val := c3 + vinfArr
							if val < mcVal {
								mcVal = val
								mcJGA = arrivalDT
								mcC3 = c3
								mcVinfJGA = vInfDepJup
								mcVinfPluto = vinfArr
								mcLaunch = launchDT
								mcPCE = plutoArrivalDT
							}
						}
					}
				}
			}
		}
	}
	fmt.Printf("maxRp = %f km (%f radii)\n", maxRp, maxRp/smd.Jupiter.Radius)
	fmt.Printf("=== MIN ===\nDT: %s\tc3=%.3f km^2/s^2\n", minEarthLaunchDT, minC3)
	fmt.Printf("min arrival at Pluto: %s\n", minPlutoArrivalDT)
	fmt.Printf("min vInf at Pluto: %f km/s\n", minVinfInPluto)
	fmt.Printf("=== SELECTION ===\nJGA: %s\tc3=%.3f km^2/s^2\nvInf@JGA: %.3f km/s\tvInf@PCE: %.3f\nLaunch: %s\tPluto arrival: %s\n", mcJGA, mcC3, mcVinfJGA, mcVinfPluto, mcLaunch, mcPCE)
}

func pcpGenerator(initPlanet, arrivalPlanet smd.CelestialObject, initLaunch, maxLaunch, initArrival, maxArrival time.Time, ptsPerLaunchDay, ptsPerArrivalDay float64, plotC3 bool, pcpName string, verbose bool) (c3Map, tofMap, vinfMap map[time.Time][]float64, vInfInitVecs, vInfArriVecs map[time.Time][]mat64.Vector) {
	launchWindow := int(maxLaunch.Sub(initLaunch).Hours() / 24)    //days
	arrivalWindow := int(maxArrival.Sub(initArrival).Hours() / 24) //days
	// Create the output arrays
	c3Map = make(map[time.Time][]float64)
	tofMap = make(map[time.Time][]float64)
	vinfMap = make(map[time.Time][]float64)
	vInfInitVecs = make(map[time.Time][]mat64.Vector)
	vInfArriVecs = make(map[time.Time][]mat64.Vector)
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
		c3Map[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		tofMap[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		vinfMap[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		vInfInitVecs[launchDT] = make([]mat64.Vector, arrivalWindow*int(ptsPerArrivalDay))
		vInfArriVecs[launchDT] = make([]mat64.Vector, arrivalWindow*int(ptsPerArrivalDay))

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
				// Store a nil vector to not loose track of indexing
				vInfInitVecs[launchDT][arrivalIdx] = *mat64.NewVector(3, nil)
				vInfArriVecs[launchDT][arrivalIdx] = *mat64.NewVector(3, nil)
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
				VInfArrival.SubVec(arrivalV, Vf)
				vInfArrival = mat64.Norm(VInfArrival, 2)
				vInfInitVecs[launchDT][arrivalIdx] = *VInfInit
				vInfArriVecs[launchDT][arrivalIdx] = *VInfArrival
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
