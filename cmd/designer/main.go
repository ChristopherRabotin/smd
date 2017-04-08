package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

const (
	defaultScenario = "~~unset~~"
	dateTimeFormat  = "2006-01-02 15:04:05"
)

var (
	scenario               string
	numCPUs                int
	initLaunch, maxArrival time.Time
	periapsisRadii         []float64
	planets                []smd.CelestialObject
	maxDeltaVs             []float64
	maxC3, maxVinfArrival  float64
	cpuChan                chan (bool)
	rsltChan               chan (Result)
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
	flag.IntVar(&numCPUs, "cpus", -1, "number of CPUs to use for after first finding (set to 0 for max CPUs)")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}
	availableCPUs := runtime.NumCPU()
	if numCPUs <= 0 || numCPUs > availableCPUs {
		numCPUs = availableCPUs
	}
	runtime.GOMAXPROCS(numCPUs)
	fmt.Printf("running on %d CPUs\n", numCPUs)

	cpuChan = make(chan (bool), numCPUs)
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
	if verbose {
		log.Printf("[info] searching for %s -> %s", smd.Earth.Name, planets[0].Name)
	}
	c3Map, tofMap, _, _, vInfArriVecs := smd.PCPGenerator(smd.Earth, planets[0], initLaunch, maxArrival, initLaunch, maxArrival, 1, 1, true, false, false)
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
			// Fulfills the launch requirements.
			rVec, _ := vInfArriVecs[launchDT][arrivalIdx].Dims()
			if rVec == 0 {
				log.Printf("WTF?! [%s][%d] arrival vector is empty?!\n%+v", launchDT, arrivalIdx, mat64.Formatted(&vInfArriVecs[launchDT][arrivalIdx]))
				continue
			}
			vInfIn := []float64{vInfArriVecs[launchDT][arrivalIdx].At(0, 0), vInfArriVecs[launchDT][arrivalIdx].At(1, 0), vInfArriVecs[launchDT][arrivalIdx].At(2, 0)}
			prevResult := NewResult(launchDT, c3, len(planets)-1)
			cpuChan <- true
			go GAPCP(arrivalDT, 0, vInfIn, prevResult)
		}
	}
}

// GAPCP performs the recursion.
func GAPCP(launchDT time.Time, planetNo int, vInfIn []float64, prevResult Result) {
	isLastPlanet := planetNo == len(planets)-2
	log.Printf("[info] searching for %s -> %s", planets[planetNo].Name, planets[planetNo+1].Name)
	vinfDep, tofMap, vinfArr, vinfMapVecs, vInfNextInVecs := smd.PCPGenerator(planets[planetNo], planets[planetNo+1], launchDT, launchDT.Add(24*time.Hour), launchDT, maxArrival, 1, 1, false, false, false)
	// Go through solutions and move on with values which are within the constraints.
	vInfInNorm := smd.Norm(vInfIn)
	minRp := periapsisRadii[planetNo]
	maxDV := maxDeltaVs[planetNo]
	for depDT, vInfDepPerDay := range vinfDep {
		for arrIdx, vInfDep := range vInfDepPerDay {
			flybyDV := math.Abs(vInfInNorm - vInfDep)
			if flybyDV < maxDV {
				log.Println("[debug] valid delta-V")
				// Check if the rP is okay
				vInfOut := []float64{vinfMapVecs[depDT][arrIdx].At(0, 0), vinfMapVecs[depDT][arrIdx].At(1, 0), vinfMapVecs[depDT][arrIdx].At(2, 0)}
				_, rp, _, _, _, _ := smd.GAFromVinf(vInfIn, vInfOut, smd.Jupiter)
				if minRp > 0 && rp < minRp {
					log.Printf("[debug] rP no good (%f km)", rp)
					continue // Too close, ignore
				}
				TOF := tofMap[depDT][arrIdx]
				arrivalDT := launchDT.Add(time.Duration(TOF*24) * time.Hour)
				if isLastPlanet {
					log.Println("[debug] IS last planet")
					vinfArr := vinfArr[depDT][arrIdx]
					if vinfArr < maxVinfArrival {
						log.Println("[debug] valid traj!")
						// This is a valid trajectory?
						// Add information to result.
						result := prevResult.Clone()
						result.arrival = arrivalDT
						result.vInf = vinfArr
						rsltChan <- result
					}
					// All done, let's free that CPU
					<-cpuChan
				} else {
					log.Println("[debug] not last planet")
					result := prevResult.Clone()
					result.flybys = append(result.flybys, GAResult{arrivalDT, flybyDV, rp})
					// Recursion
					vInfInNext := []float64{vInfNextInVecs[depDT][arrIdx].At(0, 0), vInfNextInVecs[depDT][arrIdx].At(1, 0), vInfNextInVecs[depDT][arrIdx].At(2, 0)}
					GAPCP(arrivalDT, planetNo+1, vInfInNext, result)
				}
			} else {
				log.Printf("[debug] delta-V too big (%f)", flybyDV)
				// Won't go anywhere, let's move onto another date.
				<-cpuChan
			}
		}
	}
}
