package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/spf13/viper"
)

const (
	defaultScenario = "~~unset~~"
	dateTimeFormat  = "2006-01-02 15:04:05"
	ultraDebug      = true
)

var (
	scenario string
	numCPUs  int
	arrival  Arrival
	flybys   []Flyby
	cpuChan  chan (bool)
	rsltChan chan (Result)
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
	flag.IntVar(&numCPUs, "cpus", -1, "number of CPUs to use for after first finding (set to 0 for max CPUs)")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// End profiling
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}
	scenario = strings.Replace(scenario, ".toml", "", 1)
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
	prefix := viper.GetString("general.fileprefix")
	verbose := viper.GetBool("general.verbose")
	if verbose {
		log.Printf("[conf] file prefix: %s\n", prefix)
	}
	timeStep := viper.GetDuration("general.step")
	if verbose {
		log.Printf("[conf] time step: %s\n", timeStep)
	}

	launch := readLaunch()
	arrival = readArrival()
	flybys = readAllFlybys(launch.from, arrival.until)

	if verbose {
		log.Printf("[conf] Launch: %s", launch)
		for no, fb := range flybys {
			log.Printf("[conf] Flyby#%d %s", no, fb)
		}
		log.Printf("[conf] Arrival: %s", arrival)
	}

	// Starting the streamer
	rsltChan = make(chan (Result), 10) // Buffered to not loose any data.
	planets := make([]smd.CelestialObject, len(flybys)+1)
	for i, fb := range flybys {
		planets[i] = fb.planet
	}
	planets[len(flybys)] = arrival.planet
	go StreamResults(prefix, planets, rsltChan)

	// Let's do the magic.
	if verbose {
		log.Printf("[info] searching for %s -> %s", launch.planet.Name, flybys[0].planet.Name)
	}
	c3Map, tofMap, _, _, vInfArriVecs := smd.PCPGenerator(launch.planet, flybys[0].planet, launch.from, launch.until, flybys[0].from, flybys[0].until, 1, 1, true, ultraDebug, false)
	/*for initLaunch.Before(maxArrival) {
		smd.FreeEphemeralData(smd.Earth, initLaunch.Year())
		smd.FreeEphemeralData(planets[0], initLaunch.Year())
		initLaunch = initLaunch.AddDate(1, 0, 0)
	}*/
	if *cpuprofile != "" {
		return
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
	for launchDT, c3PerDay := range c3Map {
		for arrivalIdx, c3 := range c3PerDay {
			if c3 > launch.maxC3 || c3 == 0 {
				if ultraDebug {
					log.Printf("[debug] c3 not good (%f)", c3)
				}
				continue // Cannot use this launch
			}
			arrivalTOF := tofMap[launchDT][arrivalIdx]
			arrivalDT := launchDT.Add(time.Duration(arrivalTOF*24) * time.Hour)
			if arrivalDT.After(flybys[0].until) {
				if ultraDebug {
					log.Printf("[debug] DT not good (%s)", arrivalDT)
				}
				continue
			}
			vInfInVec := vInfArriVecs[launchDT][arrivalIdx]
			if r, _ := vInfInVec.Dims(); r == 0 {
				continue
			}
			// Fulfills the launch requirements.
			vInfIn := []float64{vInfInVec.At(0, 0), vInfInVec.At(1, 0), vInfInVec.At(2, 0)}
			prevResult := NewResult(launchDT, c3, len(planets)-1)
			cpuChan <- true
			go GAPCP(arrivalDT, 0, vInfIn, prevResult)
		}
	}
}

// GAPCP performs the recursion.
func GAPCP(launchDT time.Time, planetNo int, vInfIn []float64, prevResult Result) {
	inFlyby := flybys[planetNo]
	minRp := inFlyby.minPeriapsisRadius
	maxDV := inFlyby.maxDeltaV
	fromPlanet := inFlyby.planet
	var toPlanet smd.CelestialObject
	var minArrival, maxArrival time.Time
	var isLastPlanet bool
	if planetNo+1 == len(flybys) {
		isLastPlanet = true
		toPlanet = arrival.planet
		minArrival = arrival.from
		maxArrival = arrival.until
	} else {
		isLastPlanet = false // Not needed, only for clarity
		next := flybys[planetNo+1]
		toPlanet = next.planet
		minArrival = next.from
		maxArrival = next.until
	}
	log.Printf("[info] searching for %s -> %s (last? %v)", fromPlanet.Name, toPlanet.Name, isLastPlanet)
	vinfDep, tofMap, vinfArr, vinfMapVecs, vInfNextInVecs := smd.PCPGenerator(fromPlanet, toPlanet, launchDT, launchDT.Add(24*time.Hour), minArrival, maxArrival, 1, 1, false, false, false)
	// Go through solutions and move on with values which are within the constraints.
	vInfInNorm := smd.Norm(vInfIn)
	for depDT, vInfDepPerDay := range vinfDep {
		for arrIdx, vInfOutNorm := range vInfDepPerDay {
			vInfOut := []float64{vinfMapVecs[depDT][arrIdx].At(0, 0), vinfMapVecs[depDT][arrIdx].At(1, 0), vinfMapVecs[depDT][arrIdx].At(2, 0)}
			flybyDV := math.Abs(vInfInNorm - vInfOutNorm)
			if (maxDV > 0 && flybyDV < maxDV) || maxDV == 0 {
				log.Printf("[debug] dv OK (%f km/s)", flybyDV)
				// Check if the rP is okay
				_, rp, _, _, _, _ := smd.GAFromVinf(vInfIn, vInfOut, fromPlanet)
				if minRp > 0 && rp < minRp {
					log.Printf("[debug] rP no good (%f km)", rp)
					continue // Too close, ignore
				}
				TOF := tofMap[depDT][arrIdx]
				arrivalDT := launchDT.Add(time.Duration(TOF*24) * time.Hour)
				result := prevResult.Clone()
				rslt := GAResult{launchDT, flybyDV, rp}
				result.flybys = append(result.flybys, rslt)
				if isLastPlanet {
					vinfArr := vinfArr[depDT][arrIdx]
					if vinfArr < arrival.maxVinf {
						log.Println("[debug] valid traj!")
						// This is a valid trajectory?
						// Add information to result.
						result.arrival = arrivalDT
						result.vInf = vinfArr
						rsltChan <- result
					} else {
						log.Printf("[debug] vInf too high (%f km/s)", vinfArr)
					}
					// All done, let's free that CPU
					<-cpuChan
				} else {
					// Recursion
					vInfInNext := []float64{vInfNextInVecs[depDT][arrIdx].At(0, 0), vInfNextInVecs[depDT][arrIdx].At(1, 0), vInfNextInVecs[depDT][arrIdx].At(2, 0)}
					GAPCP(arrivalDT, planetNo+1, vInfInNext, result)
				}
			} else {
				// Won't go anywhere, let's move onto another date. and clear queue if needed.
				select {
				case <-cpuChan:
					continue
				default:
					continue
				}
			}
		}
	}
}
