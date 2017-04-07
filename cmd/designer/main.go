package main

import (
	"flag"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

const (
	defaultScenario = "~~unset~~"
	dateTimeFormat  = "2006-01-02 15:04:05"
)

var (
	scenario               string
	initLaunch, maxArrival time.Time
	periapsisRadii         []float64
	planets                []smd.CelestialObject
	maxDeltaVs             []float64
	maxC3, maxVinfArrival  float64
	rsltChan               chan (Result)
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}
	// Load scenario
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("./%s.toml not found", scenario)
	}
	// Read scenario
	prefix := viper.GetString("General.fileprefix")
	verbose := viper.GetBool("General.verbose")
	if verbose {
		log.Printf("[info] file prefix: %s\n", prefix)
	}
	timeStepStr := viper.GetString("General.step")
	timeStep, durErr := time.ParseDuration(timeStepStr)
	if durErr != nil {
		log.Fatalf("could not understand `step`: %s", durErr)
	}
	if verbose {
		log.Printf("[info] time step: %s\n", timeStep)
	}
	// Date time information
	var perr error
	initLaunchJD := viper.GetFloat64("General.from")
	if initLaunchJD == 0 {
		initLaunch, perr = time.Parse(dateTimeFormat, viper.GetString("General.from"))
		if perr != nil {
			log.Fatalf("could not understand `from`: %s", perr)
		}
	} else {
		initLaunch = julian.JDToTime(initLaunchJD)
	}
	if verbose {
		log.Printf("[info] init launch: %s\n", initLaunch)
	}
	maxArrivalJD := viper.GetFloat64("General.until")
	if maxArrivalJD == 0 {
		maxArrival, perr = time.Parse(dateTimeFormat, viper.GetString("General.until"))
		if perr != nil {
			log.Fatalf("could not understand `until`: %s", perr)
		}
	} else {
		maxArrival = julian.JDToTime(maxArrivalJD)
	}
	if verbose {
		log.Printf("[info] max arrival: %s\n", maxArrival)
	}
	// Read all the planets.
	planetSlice := viper.GetStringSlice("General.planets")
	planets = make([]smd.CelestialObject, len(planetSlice))
	periapsisRadii = make([]float64, len(planetSlice))
	maxDeltaVs = make([]float64, len(planetSlice))
	for pNo, planetStr := range planetSlice {
		planet, err := smd.CelestialObjectFromString(planetStr)
		if err != nil {
			log.Fatalf("could not read planet #%d: %s", pNo, err)
		}
		planets[pNo] = planet
	}
	// Read and compute the radii constraints
	for pNo, periRfactorStr := range viper.GetStringSlice("General.periRFactor") {
		periRfactor, err := strconv.ParseFloat(periRfactorStr, 64)
		if err != nil {
			log.Fatalf("could not read radius periapsis factor #%d: %s", pNo, err)
		}
		periapsisRadii[pNo] = periRfactor * planets[pNo].Radius
	}
	// Read the deltaV constraints
	for pNo, deltaVStr := range viper.GetStringSlice("General.maxDeltaV") {
		deltaV, err := strconv.ParseFloat(deltaVStr, 64)
		if err != nil {
			log.Fatalf("could not read maximum deltaV #%d: %s", pNo, err)
		}
		maxDeltaVs[pNo] = deltaV
	}
	// Now summarize the planet passages
	if verbose {
		for pNo, planet := range planets {
			if pNo != len(planets)-1 {
				log.Printf("[info] #%d: %s\trP: %f km\tdeltaV: %f km/s\n", pNo, planet.Name, periapsisRadii[pNo], maxDeltaVs[pNo])
			} else {
				log.Printf("[info] #%d: %s (destination)\n", pNo, planet.Name)
			}
		}
	}
	// Read departure/arrival constraints.
	if viper.IsSet("DepartureConstraints.c3") {
		maxC3 = viper.GetFloat64("DepartureConstraints.c3")
	}
	if verbose {
		if maxC3 > 0 {
			log.Printf("[info] max c3: %f km^2/s^2\n", maxC3)
		} else {
			log.Println("[warn] no max c3 set")
		}
	}
	if viper.IsSet("ArrivalConstraints.vInf") {
		maxVinfArrival = viper.GetFloat64("ArrivalConstraints.vInf")
	}
	if verbose {
		if maxVinfArrival > 0 {
			log.Printf("[info] max vInf: %f km/s\n", maxVinfArrival)
		} else {
			log.Println("[warn] no max vInf set")
		}
	}
	// Starting the streamer
	rsltChan = make(chan (Result), 10) // Buffered to not loose any data.
	go StreamResults(prefix, planets, rsltChan)

	// Let's do the magic.
	// Always leave Earth.
	// NOTE: This is a VERY broad sweep.
	c3Map, tofMap, _, _, vInfArriVecs := smd.PCPGenerator(smd.Earth, planets[0], initLaunch, maxArrival, initLaunch, maxArrival, 1, 1, true, true)
	for launchDT, c3PerDay := range c3Map {
		for arrivalIdx, c3 := range c3PerDay {
			if c3 > maxC3 {
				continue // Cannot use this launch
			}
			arrivalTOF := tofMap[launchDT][arrivalIdx]
			arrivalDT := launchDT.Add(time.Duration(arrivalTOF*24) * time.Hour)
			if arrivalDT.After(maxArrival) {
				continue
			}
			// This seems to work.
			vInfIn := []float64{vInfArriVecs[launchDT][arrivalIdx].At(0, 0), vInfArriVecs[launchDT][arrivalIdx].At(1, 0), vInfArriVecs[launchDT][arrivalIdx].At(2, 0)}
			prevResult := NewResult(launchDT, c3, len(planets)-1)
			GAPCP(launchDT, 0, vInfIn, prevResult)
		}
	}
}

// GAPCP performs the recursion.
func GAPCP(launchDT time.Time, planetNo int, vInfIn []float64, prevResult Result) {
	isLastPlanet := planetNo == len(planets)-1
	vinfDep, tofMap, vinfArr, vinfMapVecs, vInfNextInVecs := smd.PCPGenerator(smd.Jupiter, smd.Pluto, launchDT, launchDT.Add(24*time.Hour), launchDT, maxArrival, 1, 1, false, false)
	// Go through solutions and move on with values which are within the constraints.
	vInfInNorm := smd.Norm(vInfIn)
	minRp := periapsisRadii[planetNo]
	maxDV := maxDeltaVs[planetNo]
	for depDT, vInfDepPerDay := range vinfDep {
		for arrIdx, vInfDep := range vInfDepPerDay {
			flybyDV := math.Abs(vInfInNorm - vInfDep)
			if flybyDV < maxDV {
				// Check if the rP is okay
				vInfOut := []float64{vinfMapVecs[depDT][arrIdx].At(0, 0), vinfMapVecs[depDT][arrIdx].At(1, 0), vinfMapVecs[depDT][arrIdx].At(2, 0)}
				_, rp, _, _, _, _ := smd.GAFromVinf(vInfIn, vInfOut, smd.Jupiter)
				if minRp > 0 && rp < minRp {
					continue // Too close, ignore
				}
				TOF := tofMap[depDT][arrIdx]
				arrivalDT := launchDT.Add(time.Duration(TOF*24) * time.Hour)
				if isLastPlanet {
					vinfArr := vinfArr[depDT][arrIdx]
					if vinfArr < maxVinfArrival {
						// This is a valid trajectory?
						// Add information to result.
						result := prevResult.Clone()
						result.arrival = arrivalDT
						result.vInf = vinfArr
						rsltChan <- result
					}
				} else {
					result := prevResult.Clone()
					result.flybys = append(result.flybys, GAResult{arrivalDT, flybyDV, rp})
					// Recursion
					vInfInNext := []float64{vInfNextInVecs[depDT][arrIdx].At(0, 0), vInfNextInVecs[depDT][arrIdx].At(1, 0), vInfNextInVecs[depDT][arrIdx].At(2, 0)}
					GAPCP(launchDT, planetNo+1, vInfInNext, result)
				}
			}
		}
	}
}
