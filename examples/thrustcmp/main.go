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
	wg          sync.WaitGroup
	departEarth = true
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

func createSpacecraft(thruster thrusterType, numThrusters int, dist float64) (*smd.Spacecraft, float64) {
	/* Building spacecraft */
	//thrusters := []smd.Thruster{&smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}}
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
	return smd.NewSpacecraft(name, dryMass, fuelMass, smd.NewUnlimitedEPS(), thrusters, false, []*smd.Cargo{}, []smd.Waypoint{smd.NewReachDistance(dist, true, nil)}), thrust
}

func main() {
	wg.Add(1)
	var fn string
	if departEarth {
		fn = "gto2escape-argperi"
	}
	rslts := make(chan string, 10)
	go streamResults(fn, rslts)

	aGTO, eGTO := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)

	for ω := 0.0; ω < 360; ω += 5.0 {
		fmt.Printf("\n##### %.1f deg #####\n", ω)
		for _, thruster := range []thrusterType{pps1350, pps5000, bht1500, bht8000, hermes, vx200} {
			for numThrusters := 1; numThrusters <= 4; numThrusters++ {
				var initOrbit *smd.Orbit
				if departEarth {
					initOrbit = smd.NewOrbitFromOE(aGTO, eGTO, 31, 330, ω, 210, smd.Earth)
				}
				sc, maxThrust := createSpacecraft(thruster, numThrusters, smd.Earth.SOI)
				initV := initOrbit.VNorm()
				initFuel := sc.FuelMass
				startDT := time.Now()
				// Propagate
				astro := smd.NewPreciseMission(sc, initOrbit, startDT, startDT.Add(-1), smd.Cartesian, smd.Perturbations{}, time.Minute, smd.ExportConfig{})
				astro.Propagate()
				// Compute data.
				tof := astro.CurrentDT.Sub(startDT).Hours() / 24
				deltaV := astro.Orbit.VNorm() - initV
				deltaFuel := sc.FuelMass - initFuel
				// Convert to heliocentric
				astro.Orbit.ToXCentric(smd.Sun, astro.CurrentDT)
				vInf := astro.Orbit.VNorm() - smd.Earth.HelioOrbit(astro.CurrentDT).VNorm()
				rslts <- fmt.Sprintf("%.3f,%.3f,%.3f,%.3f,%.3f,%.3f\n", ω, maxThrust, tof, deltaV, deltaFuel, vInf)
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
	f.WriteString("arg. peri (degrees), thrust (N), T.O.F. (days), \\Delta V (km/s), fuel (kf), \\V_{\\infty} (km/s)\n")
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
