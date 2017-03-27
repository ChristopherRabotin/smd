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
)

var (
	wg             sync.WaitGroup
	departEarth    = true
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
	return smd.NewSpacecraft(name, dryMass, fuelMass, smd.NewUnlimitedEPS(), thrusters, false, []*smd.Cargo{}, []smd.Waypoint{smd.NewReachDistance(dist, further, nil)}), thrust
}

func main() {
	wg.Add(1)
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
	rslts := make(chan string, 10)
	go streamResults(fn, rslts)

	aGTO, eGTO := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	aMRO, eMRO := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	startDT := time.Now()
	earthOrbit := smd.Earth.HelioOrbit(startDT)
	marsOrbit := smd.Mars.HelioOrbit(startDT)

	numThrusters := 1

	for _, thruster := range []thrusterType{pps1350, pps5000, bht1500, bht8000, hermes, vx200} {

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
			if departEarth {
				sc.DryMass = 900 + 577 + 1e3 + 1e3 // Add MRO, Curiosity and Schiaparelli, and suppose 1 ton bus.
				sc.FuelMass = 3e3
			} else {
				// Suppose less return mass
				sc.DryMass = 500 + 1e3 // Add Schiaparelli return, and suppose 1 ton bus.
				sc.FuelMass = 1e3
			}
			initV := initOrbit.VNorm()
			initFuel := sc.FuelMass
			// Propagate
			astro := smd.NewPreciseMission(sc, initOrbit, startDT, startDT.Add(-1), smd.Cartesian, smd.Perturbations{}, 5*time.Minute, smd.ExportConfig{})
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
			if interplanetary || true {
				rslts <- csv
				//break
			}
		}

		if !interplanetary && false {
			rslts <- bestCSV
		}
	}
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
