package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ChristopherRabotin/smd"
)

const (
	// TTypeAuto lets the Lambert solver determine the type
	pps1350 thrusterType = iota + 1
	pps5000
	bht1500
	bht8000
	hermes
	vx200
	opti = true
)

var (
	wg             sync.WaitGroup
	departEarth    = false
	interplanetary = false
)

type thrusterType uint8

func (tt thrusterType) Type() smd.EPThruster {
	switch tt {
	case pps1350:
		return new(smd.PPS1350)
	case pps5000:
		return new(smd.PPS5000)
	case bht1500:
		return new(smd.BHT1500)
	case bht8000:
		return new(smd.BHT8000)
	case hermes:
		return new(smd.HERMeS)
	case vx200:
		return new(smd.VX200)
	default:
		panic("unknown thruster")
	}
}

func (tt thrusterType) BestArgPeri(cluster int) float64 {
	switch tt {
	case pps1350:
		return 320 // cluster irrelevant
	case pps5000:
		return 70 // idem
	case bht1500:
		return 60 // idem
	case bht8000:
		return 230 // idem
	case hermes:
		return 110 // idem
	case vx200:
		if cluster == 1 {
			return 270
		}
		return 190
	default:
		panic("unknown thruster")
	}
}

func (tt thrusterType) String() string {
	switch tt {
	case pps1350:
		return "PPS1350"
	case pps5000:
		return "PPS5000"
	case bht1500:
		return "BHT1500"
	case bht8000:
		return "BHT8000"
	case hermes:
		return "HERMeS"
	case vx200:
		return "VX200"
	default:
		panic("unknown thruster")
	}
}

func createSpacecraft(thruster thrusterType, numThrusters int, dist float64, further bool) (*smd.Spacecraft, float64) {
	/* Building spacecraft */
	thrusters := make([]smd.EPThruster, numThrusters)
	thrust := 0.0
	for i := 0; i < numThrusters; i++ {
		thrusters[i] = thruster.Type()
		voltage, power := thruster.Type().Max()
		thisThrust, _ := thruster.Type().Thrust(voltage, power)
		thrust += thisThrust
	}
	dryMass := 1.0
	fuelMass := 1000.0
	name := fmt.Sprintf("%dx%s", numThrusters, thruster)
	fmt.Printf("\n===== %s ======\n", name)
	waypoints := []smd.Waypoint{smd.NewReachDistance(dist, further, nil)}
	if opti {
		if interplanetary {
			if departEarth {
				waypoints = []smd.Waypoint{smd.NewOrbitTarget(smd.Mars.HelioOrbit(time.Now()), nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL), smd.NewCruiseToDistance(dist, further, nil)}
			} else {
				waypoints = []smd.Waypoint{smd.NewOrbitTarget(smd.Earth.HelioOrbit(time.Now()), nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL, smd.OptiΔeCL), smd.NewCruiseToDistance(dist, further, nil)}
			}
		} else {
			if departEarth {
				// Create virtual orbit
				tgt := smd.NewOrbitFromOE(smd.Earth.SOI, 0.75, 0, 0, 230, 0, smd.Earth)
				waypoints = []smd.Waypoint{smd.NewOrbitTarget(*tgt, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔeCL), smd.NewCruiseToDistance(dist, further, nil)}
			} else {
				tgt := smd.NewOrbitFromOE(smd.Mars.SOI, 0.85, 0, 0, 230, 0, smd.Mars)
				waypoints = []smd.Waypoint{smd.NewOrbitTarget(*tgt, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔeCL), smd.NewCruiseToDistance(dist, further, nil)}
			}
		}
		//eGTO = 0.745230 eMRO=0.852192

	}
	return smd.NewSpacecraft(name, dryMass, fuelMass, smd.NewUnlimitedEPS(), thrusters, false, []*smd.Cargo{}, waypoints), thrust
}

func main() {
	// Go through all combinations
	combinations := []struct {
		missionNo, numThrusters int
	}{{1, 1}, {1, 2}, {2, 1}, {2, 2}, {3, 8}, {3, 12}}
	for _, combin := range combinations {
		run(combin.missionNo, combin.numThrusters)
	}
	wg.Wait()
}

func run(missionNo, numThrusters int) {
	var fn string
	if departEarth {
		if interplanetary {
			fn = "earth2mars-tof"
		} else {
			fn = "gto2escape-tof"
		}
	} else {
		if interplanetary {
			fn = "mars2earth-tof"
		} else {
			fn = "mro2escape-tof"
		}
	}
	if missionNo < 3 {
		if numThrusters == 1 {
			fn += fmt.Sprintf("-%da", missionNo)
		} else {
			fn += fmt.Sprintf("-%db", missionNo)
		}
	} else if numThrusters == 8 {
		fn += "-3a"
	} else {
		fn += "-3b"
	}
	if opti {
		fn += "-opti-we"
	} else {
		fn += "-notopti"
	}
	rslts := make(chan string, 10)
	wg.Add(1)
	go streamResults(fn, rslts)

	aGTO, eGTO := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	aMRO, eMRO := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	startDT := time.Now()
	earthOrbit := smd.Earth.HelioOrbit(startDT)
	marsOrbit := smd.Mars.HelioOrbit(startDT)

	combinations := []thrusterType{ /*pps1350, pps5000, bht1500,*/ bht8000 /* hermes, vx200*/}
	if missionNo == 3 {
		combinations = []thrusterType{ /*pps5000,*/ bht8000 /*, hermes, vx200*/}
	}

	for _, thruster := range combinations {
		var bestCSV string
		var bestTOF = 1e9
		for ω := 0.0; ω < 360; ω += 10.0 {
			fmt.Printf("\n##### %.1f deg #####\n", ω)
			var initOrbit *smd.Orbit
			var distance float64
			var further bool
			if departEarth {
				if interplanetary {
					ugh := smd.Earth.HelioOrbit(startDT)
					initOrbit = &ugh
					further = true
					distance = marsOrbit.RNorm()
				} else {
					initOrbit = smd.NewOrbitFromOE(aGTO, eGTO, 0.0, 0, ω, 210, smd.Earth)
					distance = smd.Earth.SOI
					further = true
				}
			} else {
				if interplanetary {
					ugh := smd.Mars.HelioOrbit(startDT)
					initOrbit = &ugh
					further = false
					distance = earthOrbit.RNorm()
				} else {
					initOrbit = smd.NewOrbitFromOE(aMRO, eMRO, 0.0, 0, ω, 180, smd.Mars)
					distance = smd.Mars.SOI
					further = true
				}
			}
			sc, maxThrust := createSpacecraft(thruster, numThrusters, distance, further)
			if missionNo == 2 {
				if departEarth {
					sc.DryMass = 900 + 577 + 1e3 + 1e3 // Add MRO, Curiosity and Schiaparelli, and suppose 1 ton bus.
					sc.FuelMass = 3e3
				} else {
					// Suppose less return mass
					sc.DryMass = 500 + 1e3 // Add Schiaparelli return, and suppose 1 ton bus.
					sc.FuelMass = 1e3
				}
			} else if missionNo == 3 {
				sc.DryMass = 52e3
				if departEarth {
					sc.FuelMass = 24e3
				} else {
					sc.FuelMass = 6e3
				}
			}
			initV := initOrbit.VNorm()
			initFuel := sc.FuelMass
			// Propagate
			ts := 5 * time.Minute
			export := smd.ExportConfig{Filename: fn + sc.Name, AsCSV: true, Cosmo: true, Timestamp: false}
			endDT := startDT.Add(-1)
			if opti {
				//	endDT = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			astro := smd.NewPreciseMission(sc, initOrbit, startDT, endDT, smd.Perturbations{}, ts, false, export)
			astro.Propagate()
			// Compute data.
			tof := astro.CurrentDT.Sub(startDT).Hours() / 24
			deltaV := astro.Orbit.VNorm() - initV
			deltaFuel := sc.FuelMass - initFuel
			var vInf float64
			if departEarth {
				if interplanetary { // Arriving at Mars, check how fast we're going compared to some standard velocity
					vInf = astro.Orbit.VNorm() - marsOrbit.VNorm()
				} else {
					astro.Orbit.ToXCentric(smd.Sun, astro.CurrentDT)
					vInf = astro.Orbit.VNorm() - smd.Earth.HelioOrbit(astro.CurrentDT).VNorm()
				}
			} else {
				if interplanetary {
					vInf = astro.Orbit.VNorm() - earthOrbit.VNorm()
				} else {
					astro.Orbit.ToXCentric(smd.Sun, astro.CurrentDT)
					vInf = astro.Orbit.VNorm() - smd.Mars.HelioOrbit(astro.CurrentDT).VNorm()
				}
			}

			csv := fmt.Sprintf("%.3f,%s,%.3f,%.3f,%.3f,%.3f,%.3f\n", ω, sc.Name, maxThrust, tof, deltaV, deltaFuel, vInf)
			// Check if best
			if tof < bestTOF {
				bestTOF = tof
				bestCSV = csv
			}
			if true {
				rslts <- csv
				break
			}
		}

		if !interplanetary && false {
			rslts <- bestCSV
		}
	}
	close(rslts)
}

func streamResults(fn string, rslts <-chan string) {
	// Write CSV file.
	f, err := os.Create(fmt.Sprintf("./%s.csv", fn))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	// Header
	f.WriteString("arg. periapsis (deg), name, thrust (N), T.O.F. (days), \\Delta V (km/s), fuel (kf), \\V_{\\infty} (km/s)\n")
	for {
		rslt, more := <-rslts
		if more {
			if _, err := f.WriteString(rslt); err != nil {
				panic(err)
			}
		} else {
			break
		}
	}
	f.Close()
	wg.Done()
}
