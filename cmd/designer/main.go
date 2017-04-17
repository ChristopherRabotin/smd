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
	"sync"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/spf13/viper"
)

const (
	defaultScenario    = "~~unset~~"
	dateFormat         = "2006-01-02 15:04:05"
	dateFormatFilename = "2006-01-02-15.04.05"
)

var (
	wg         sync.WaitGroup
	scenario   string
	prefix     string
	outputdir  string
	timeStep   time.Duration
	numCPUs    int
	ultraDebug bool
	arrival    Arrival
	flybys     []Flyby
	cpuChan    chan (bool)
	rsltChan   chan (Result)
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
	flag.IntVar(&numCPUs, "cpus", -1, "number of CPUs to use for after first finding (set to 0 for max CPUs)")
	flag.BoolVar(&ultraDebug, "debug", false, "debug everything (really verbose)")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if ultraDebug {
		log.Println("[info] DEBUG is ON")
	} else {
		log.Println("[info] DEBUG is OFF")
	}
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
	log.Printf("[info] running on %d CPUs\n", numCPUs)

	cpuChan = make(chan (bool), numCPUs)
	// Load scenario
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("./%s.toml: Error %s", scenario, err)
	}
	// Read scenario
	prefix = viper.GetString("general.fileprefix")
	outputdir = viper.GetString("general.outputdir")
	if len(outputdir) == 0 {
		outputdir = "./"
	}
	verbose := viper.GetBool("general.verbose")
	if verbose {
		log.Printf("[conf] file prefix: %s\n", prefix)
		log.Printf("[conf] file output: %s\n", outputdir)
	}
	timeStep = viper.GetDuration("general.step")
	if verbose {
		log.Printf("[conf] time step: %s\n", timeStep)
	}

	launch := readLaunch()
	arrival = readArrival()
	flybys = readAllFlybys(launch.from, arrival.until)

	if verbose {
		log.Printf("[conf] Launch: %s", launch)
		for no, fb := range flybys {
			log.Printf("[conf] Flyby#%d %s", no+1, fb)
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
	c3Map, tofMap, _, _, vInfArriVecs := smd.PCPGenerator(launch.planet, flybys[0].planet, launch.from, launch.until, flybys[0].from, flybys[0].until, 1, 1, smd.TTypeAuto, true, ultraDebug, false)
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
			wg.Add(1)
			go GAPCP(arrivalDT, flybys[0], 0, vInfIn, prevResult)
		}
	}
	log.Println("[info] All valid launches started")
	wg.Wait()
	log.Println("[info] Done")
}

// GAPCP performs the recursion.
func GAPCP(launchDT time.Time, inFlyby Flyby, planetNo int, vInfIn []float64, prevResult Result) {
	minRp := inFlyby.minPeriapsisRadius
	maxDV := inFlyby.maxDeltaV
	fromPlanet := inFlyby.planet
	var toPlanet smd.CelestialObject
	var minArrival, maxArrival time.Time
	var isLastPlanet bool
	if planetNo+1 == len(flybys) {
		if ultraDebug {
			log.Println("reached last planet")
		}
		isLastPlanet = true
		toPlanet = arrival.planet
		minArrival = arrival.from
		maxArrival = arrival.until
	} else {
		isLastPlanet = false // Only for clarity (not needed, initializes at false)
		next := flybys[planetNo+1]
		toPlanet = next.planet
		minArrival = next.from
		maxArrival = next.until
	}
	// Semi-smart memory allocation to avoid too much allocation.
	if minArrival.Before(launchDT) {
		minArrival = launchDT
	}
	if maxArrival.Before(launchDT) {
		maxArrival = launchDT
	}
	if inFlyby.isResonant {
		log.Printf("[info] searching for resonance %.1f:1 with %s (@%s)", inFlyby.resonance, fromPlanet.Name, launchDT.Format(dateFormat))
		ga2DT := launchDT.Add(time.Duration((365.242189*24*3600)*inFlyby.resonance) * time.Second)
		fromPlanetAtGA1 := fromPlanet.HelioOrbit(launchDT)
		fromPlanetAtGA2 := fromPlanet.HelioOrbit(ga2DT)
		// Find the possible vInfOut via Lambert
		ga2R := mat64.NewVector(3, fromPlanetAtGA2.R())
		arrivalWindow := int(maxArrival.Sub(minArrival).Hours() / 24)
		for arrivalDay := 0.; arrivalDay < float64(arrivalWindow); arrivalDay += timeStep.Hours() / 24 {
			nextPlanetArrivalDT := minArrival.Add(time.Duration(arrivalDay*24) * time.Hour)
			nextPlanetR := mat64.NewVector(3, toPlanet.HelioOrbit(nextPlanetArrivalDT).R())
			ViGA2, VfNext, _, _ := smd.Lambert(ga2R, nextPlanetR, nextPlanetArrivalDT.Sub(ga2DT), smd.TTypeAuto, smd.Sun)
			vInfOutGA2Vec := mat64.NewVector(3, nil)
			vInfOutGA2Vec.SubVec(ViGA2, mat64.NewVector(3, fromPlanetAtGA2.V()))
			vInfOutGA2 := []float64{vInfOutGA2Vec.At(0, 0), vInfOutGA2Vec.At(1, 0), vInfOutGA2Vec.At(2, 0)}
			// Continue resonance
			aResonance := math.Pow(smd.Sun.GM()*math.Pow(inFlyby.resonance*fromPlanetAtGA1.Period().Seconds()/(2*math.Pi), 2), 1/3.)
			VScSunNorm := math.Sqrt(smd.Sun.GM() * ((2 / fromPlanetAtGA1.RNorm()) - 1/aResonance))
			// Compute angle theta for EGA1
			vInfInGA1Norm := smd.Norm(vInfIn)
			theta := math.Acos((math.Pow(VScSunNorm, 2) - math.Pow(vInfInGA1Norm, 2) - math.Pow(fromPlanetAtGA1.VNorm(), 2)) / (-2 * vInfInGA1Norm * fromPlanetAtGA1.VNorm()))
			if ultraDebug {
				log.Printf("[info] resonance %.1f:1 with %s (@%s): theta = %f", inFlyby.resonance, fromPlanet.Name, launchDT.Format(dateFormat), smd.Rad2deg(theta))
			}
			// Compute the VNC2ECI DCMs for EGA1.
			// WARNING: We are generating the transposed DCM because it's simpler code.
			V := smd.Unit(fromPlanetAtGA1.V())
			N := smd.Unit(fromPlanetAtGA1.H())
			C := smd.Cross(V, N)
			dcmVal := make([]float64, 9)
			for i := 0; i < 3; i++ {
				dcmVal[i] = V[i]
				dcmVal[i+3] = N[i]
				dcmVal[i+6] = C[i]
			}
			transposedDCM := mat64.NewDense(3, 3, dcmVal)
			data := "psi\trP1\trP2\n"
			step := (2 * math.Pi) / 10000
			// Print when both become higher than minRadius.
			rpsOkay := false
			minDeltaRp := math.Inf(1)
			maxSumRp := 0.0
			var bestRp target
			for ψ := step; ψ < 2*math.Pi; ψ += step {
				sψ, cψ := math.Sincos(ψ)
				vInfOutEGA1VNC := []float64{vInfInGA1Norm * math.Cos(math.Pi-theta), vInfInGA1Norm * math.Sin(math.Pi-theta) * cψ, -vInfInGA1Norm * math.Sin(math.Pi-theta) * sψ}
				vInfOutGA1Eclip := smd.MxV33(transposedDCM.T(), vInfOutEGA1VNC)
				_, rP1, bT1, bR1, _, _ := smd.GAFromVinf(vInfIn, vInfOutGA1Eclip, smd.Earth)

				vInfInGA2Eclip := make([]float64, 3)
				for i := 0; i < 3; i++ {
					vInfInGA2Eclip[i] = vInfOutGA1Eclip[i] + fromPlanetAtGA1.V()[i] - fromPlanetAtGA2.V()[i]
				}
				_, rP2, bT2, bR2, _, _ := smd.GAFromVinf(vInfInGA2Eclip, vInfOutGA2, smd.Earth)
				data += fmt.Sprintf("%f\t%f\t%f\n", smd.Rad2deg(ψ), rP1, rP2)
				if !rpsOkay && rP1 > inFlyby.minPeriapsisRadius && rP2 > inFlyby.minPeriapsisRadius {
					rpsOkay = true
					if ultraDebug {
						fmt.Printf("[ ok ] ψ=%.6f\trP1=%.3f km\trP2=%.3f km\n", smd.Rad2deg(ψ), rP1, rP2)
					}
				}
				if rpsOkay {
					if math.Abs(rP1-rP2) < minDeltaRp && rP1+rP2 > maxSumRp {
						// Just reached a new high for both rPs.
						minDeltaRp = math.Abs(rP1 - rP2)
						maxSumRp = rP1 + rP2
						bestRp = target{bT1, bT2, bR1, bR2, ψ, rP1, rP2, smd.Norm(vInfIn), smd.Norm(vInfOutGA1Eclip), smd.Norm(vInfInGA2Eclip), smd.Norm(vInfOutGA2)}
					}
					if rP1 < inFlyby.minPeriapsisRadius || rP2 < inFlyby.minPeriapsisRadius {
						rpsOkay = false
						if ultraDebug {
							fmt.Printf("[NOK ] ψ=%.6f\trP1=%.3f km\trP2=%.3f km\n", smd.Rad2deg(ψ), rP1, rP2)
						}
					}
				}
			}
			if ultraDebug {
				fmt.Printf("=== Best Rp GA: %s\n", bestRp)
			}

			// Export data
			f, err := os.Create(fmt.Sprintf("%s/%s-resonance-%s-%s--to--%s.tsv", outputdir, prefix, fromPlanet.Name, launchDT.Format(dateFormatFilename), ga2DT.Format(dateFormatFilename)))
			if err != nil {
				panic(err)
			}
			f.WriteString(data)
			f.Close()

			result := prevResult.Clone()
			// Create both the first flyby for start of resonance and the ending flyby to complete the resonance
			result.flybys = append(result.flybys, GAResult{launchDT, bestRp.ega1Vout - bestRp.ega1Vin, bestRp.Rp1, -1})
			result.flybys = append(result.flybys, GAResult{ga2DT, bestRp.ega2Vout - bestRp.ega2Vin, bestRp.Rp2, bestRp.Assocψ})
			if isLastPlanet {
				vinfArr := mat64.Norm(VfNext, 2)
				if vinfArr < arrival.maxVinf {
					log.Println("[ ok ] valid traj after resonance!")
					// This is a valid trajectory
					// Add information to result.
					result.arrival = nextPlanetArrivalDT
					result.vInf = vinfArr
					rsltChan <- result
					wg.Done()
				} else if ultraDebug {
					log.Printf("[NOK ] vInf @ %s: %f km/s", toPlanet.Name, vinfArr)
				}
				// All done, let's free that CPU
				<-cpuChan
			} else {
				// Spawn the next flyby computation.
				GAPCP(ga2DT, inFlyby.PostResonance(), planetNo, vInfOutGA2, result)
			}
		}
	} else {
		log.Printf("[info] searching for %s (@%s) -> %s (@%s :: %s)", fromPlanet.Name, launchDT.Format(dateFormat), toPlanet.Name, minArrival.Format(dateFormat), maxArrival.Format(dateFormat))
		vinfDep, tofMap, vinfArr, vinfMapVecs, vInfNextInVecs := smd.PCPGenerator(fromPlanet, toPlanet, launchDT, launchDT.Add(24*time.Hour), minArrival, maxArrival, 1, 1, smd.TTypeAuto, false, ultraDebug, false)
		// Go through solutions and move on with values which are within the constraints.
		vInfInNorm := smd.Norm(vInfIn)
		if ultraDebug {
			log.Printf("[info] searching for %s (@%s) -> %s (@%s :: %s) -- %d", fromPlanet.Name, launchDT.Format(dateFormat), toPlanet.Name, minArrival.Format(dateFormat), maxArrival.Format(dateFormat), len(vinfDep))
		}
		for depDT, vInfDepPerDay := range vinfDep {
			for arrIdx, vInfOutNorm := range vInfDepPerDay {
				vInfOutVec := vinfMapVecs[depDT][arrIdx]
				if r, _ := vInfOutVec.Dims(); r == 0 || math.IsInf(vInfOutNorm, 1) || vInfOutNorm == 0 {
					if ultraDebug {
						log.Printf("[info] skipping item when searching for %s (@%s) -> %s (@%s :: %s) -- %d", fromPlanet.Name, launchDT.Format(dateFormat), toPlanet.Name, minArrival.Format(dateFormat), maxArrival.Format(dateFormat), len(vinfDep))
					}
					continue
				}
				TOF := tofMap[depDT][arrIdx]
				arrivalDT := launchDT.Add(time.Duration(TOF*24) * time.Hour)
				vInfOut := []float64{vInfOutVec.At(0, 0), vInfOutVec.At(1, 0), vInfOutVec.At(2, 0)}
				flybyDV := math.Abs(vInfInNorm - vInfOutNorm)
				if (maxDV > 0 && flybyDV < maxDV) || maxDV == 0 {
					if ultraDebug {
						log.Printf("[ ok ] dv @ %s on %s->%s: %f km/s", fromPlanet.Name, depDT, arrivalDT, flybyDV)
					}
					// Check if the rP is okay
					// NOTE: we oppose the vInf in because we are just transfering the vInfOut to the vInfIn via this recursion calling.
					vInfInBis := []float64{-vInfIn[0], -vInfIn[1], -vInfIn[2]}
					_, rp, _, _, _, _ := smd.GAFromVinf(vInfInBis, vInfOut, fromPlanet)
					if minRp > 0 && rp < minRp {
						if ultraDebug {
							log.Printf("[NOK ] rP @ %s on %s->%s: %f km", fromPlanet.Name, depDT, arrivalDT, rp)
						}
						continue // Too close, ignore
					}
					result := prevResult.Clone()
					rslt := GAResult{launchDT, flybyDV, rp, -1}
					result.flybys = append(result.flybys, rslt)
					if isLastPlanet {
						vinfArr := vinfArr[depDT][arrIdx]
						if vinfArr < arrival.maxVinf {
							log.Println("[ ok ] valid traj!")
							// This is a valid trajectory
							// Add information to result.
							result.arrival = arrivalDT
							result.vInf = vinfArr
							rsltChan <- result
							wg.Done()
						} else if ultraDebug {
							log.Printf("[NOK ] vInf @ %s on %s->%s: %f km/s", toPlanet.Name, depDT, arrivalDT, vinfArr)
						}
						// All done, let's free that CPU
						<-cpuChan
					} else {
						// Recursion, note the -1 to create the Next (since there is an inversion between planet velocity and the spacecraft vector)
						vInfInNext := []float64{vInfNextInVecs[depDT][arrIdx].At(0, 0), vInfNextInVecs[depDT][arrIdx].At(1, 0), vInfNextInVecs[depDT][arrIdx].At(2, 0)}
						GAPCP(arrivalDT, flybys[planetNo+1], planetNo+1, vInfInNext, result)
					}
				} else {
					if ultraDebug {
						log.Printf("[NOK ] dv @ %s on %s->%s: %f km/s", fromPlanet.Name, depDT, arrivalDT, flybyDV)
					}
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
}
