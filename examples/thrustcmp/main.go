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
	maximizeVinf   = true
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
			fn = "earth2mars"
		} else {
			fn = "gto2escape"
		}
	} else {
		if interplanetary {
			fn = "mars2earth"
		} else {
			fn = "mro2escape"
		}
	}
	rslts := make(chan string, 10)
	go streamResults(fn, rslts)

	aGTO, eGTO := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	aMRO, eMRO := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	startDT := time.Now()
	earthOrbit := smd.Earth.HelioOrbit(startDT)
	marsOrbit := smd.Mars.HelioOrbit(startDT)

	for _, thruster := range []thrusterType{pps1350, pps5000, bht1500, bht8000, hermes, vx200} {
		for numThrusters := 1; numThrusters <= 4; numThrusters++ {
			var bestCSV string
			var bestVinf float64
			if maximizeVinf {
				bestVinf = -1e9
			} else {
				bestVinf = 1e9
			}
			for _, i := range []float64{0, 5.16, 28.39} {
				for ω := 0.0; ω < 180; ω += 10.0 {
					fmt.Printf("\n##### %.1f / %.1f deg #####\n", i, ω)
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
							initOrbit = smd.NewOrbitFromOE(aGTO, eGTO, i, 330, ω, 210, smd.Earth)
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
							initOrbit = smd.NewOrbitFromOE(aMRO, eMRO, i, 240, ω, 180, smd.Mars)
							distance = smd.Mars.SOI
							further = true
						}
					}
					sc, maxThrust := createSpacecraft(thruster, numThrusters, distance, further)
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
							vInf = astro.Orbit.VNorm() - smd.Earth.HelioOrbit(astro.CurrentDT).VNorm()
						}
					} else {
						if interplanetary {
							vInf = astro.Orbit.VNorm() - earthOrbit.VNorm()
						} else {
							vInf = astro.Orbit.VNorm() - smd.Mars.HelioOrbit(astro.CurrentDT).VNorm()
						}
					}

					csv := fmt.Sprintf("%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f\n", i, ω, maxThrust, tof, deltaV, deltaFuel, vInf)
					// Check if best
					if (maximizeVinf && vInf > bestVinf) || (!maximizeVinf && vInf < bestVinf) {
						bestVinf = vInf
						bestCSV = csv
					}
					if interplanetary {
						rslts <- csv
						break
					}
				}
				if interplanetary {
					break // No need to go further for interplanetary because I can't alter inc or arg of periapsis
				}
			}
			if !interplanetary {
				rslts <- bestCSV
			}
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
	f.WriteString("inc. (deg), arg. periapsis (deg), name, thrust (N), T.O.F. (days), \\Delta V (km/s), fuel (kf), \\V_{\\infty} (km/s)\n")
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
