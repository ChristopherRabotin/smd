package main

import (
	"flag"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

// NOTE: This tool runs a planet to planet simulation, without any spiral, but then attempt an injection is desired.
// The number of CPUs to use is important because that's the number of goroutines which will run in parallel.

/* === CONFIG === */
var (
	numCPUs         int
	initPlanet      = smd.Earth
	destPlanet      = smd.Mars
	initLaunch      = time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC)
	launchWindow    = time.Duration(6*30.5*24) * time.Hour // Works both plus and negative
	launchTimeStep  = time.Duration(12) * time.Hour
	withInjection   = false
	missionTimeStep = time.Hour
	fuel            = 5000.0
)

/* ===  END  === */

var (
	cpuChan     chan (bool)
	resultChan  chan (result)
	threadEnded = 0
	minLaunch   = initLaunch.Add(-launchWindow)
	maxLaunch   = initLaunch.Add(launchWindow)
)

func init() {
	// Read flags
	flag.IntVar(&numCPUs, "cpus", -1, "number of CPUs to use for this simulation (set to 0 for max CPUs)")
}

func main() {
	flag.Parse()
	availableCPUs := runtime.NumCPU()
	if numCPUs <= 0 || numCPUs > availableCPUs {
		numCPUs = availableCPUs
	}
	runtime.GOMAXPROCS(numCPUs)
	fmt.Printf("running on %d CPUs\n", numCPUs)

	cpuChan = make(chan (bool), numCPUs)
	resultChan = make(chan (result), numCPUs)
	// Populate the resultChan with initial guesses.
	for i := 0; i < numCPUs; i++ {
		resultChan <- result{false, leading, initLaunch.Add(time.Duration(i) * launchTimeStep), time.Now()}
	}
	for threadEnded < numCPUs {
		cpuChan <- true
		var sc *smd.Spacecraft
		if destPlanet.Equals(smd.Mars) {
			sc = sc2Mars(fuel)
		} else {
			sc = sc2Earth(fuel)
		}
		go targeter(sc)
	}
}

func targeter(sc *smd.Spacecraft) {
	// Grab the latest result
	someResult := <-resultChan
	launchDT := someResult.launchDT
	if someResult.status == leading {
		// Decrease the launch date
		launchDT = launchDT.Add(-launchTimeStep)
	} else {
		launchDT = launchDT.Add(launchTimeStep)
	}
	// Check if the launch date is still within the bounds
	if launchDT.Before(minLaunch) || launchDT.After(maxLaunch) {
		threadEnded++
		<-cpuChan
		return
	}
	launchOrbit := initPlanet.HelioOrbit(launchDT)
	astro := smd.NewPreciseMission(sc, &launchOrbit, launchDT, launchDT.Add(-1), smd.Cartesian, smd.Perturbations{}, missionTimeStep, smd.ExportConfig{})
	astro.Propagate()
	// Let's check if we are within the SOI of the destination planet
	scR := astro.Orbit.R()
	destOrbit := destPlanet.HelioOrbit(astro.CurrentDT)
	destR := destOrbit.R()
	deltaR := 0.
	for i := 0; i < 3; i++ {
		deltaR += math.Pow(scR[i]-destR[i], 2)
	}
	success := math.Sqrt(deltaR) < destPlanet.SOI
	// Determine whether leading or trailing
	tmpOrbit := smd.NewOrbitFromRV(scR, destOrbit.V(), smd.Sun)
	_, _, _, _, _, νSC, _, _, _ := tmpOrbit.Elements()
	_, _, _, _, _, νDest, _, _, _ := destOrbit.Elements()
	var status positionsStatus
	if νSC > νDest {
		status = leading
	} else {
		status = trailing
	}
	if success {
		// Immediately stop everything and print the success
		fmt.Printf("\n\n======\nSUCCESS!!\n\n%s\n\n======\n", someResult)
		threadEnded = numCPUs
	}
	rslt := result{succeeded: success, status: status, launchDT: launchDT, arrivalDT: astro.CurrentDT}
	resultChan <- rslt
	<-cpuChan
}

func sc2Mars(fuel float64) *smd.Spacecraft {
	// Approximate planetary distance
	marsOrbit := smd.Mars.HelioOrbit(time.Now())
	distance := marsOrbit.RNorm()
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := fuel
	ref2Mars := &smd.WaypointAction{Type: smd.REFMARS, Cargo: nil}
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	var i float64 = 61
	var Ω float64 = 240
	var ν float64 = 180
	hyper := smd.NewOrbitFromOE(a, e, i, Ω, 60, ν, smd.Mars)
	if withInjection {
		return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
			[]smd.Waypoint{
				smd.NewReachDistance(distance, true, ref2Mars),
				smd.NewLoiter(time.Hour, nil),
				smd.NewToElliptical(nil),
				smd.NewOrbitTarget(*hyper, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL),
				smd.NewLoiter(7*24*time.Hour, nil),
			})
	}
	return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{
			smd.NewReachDistance(distance, true, nil),
			smd.NewLoiter(time.Hour, nil),
		})
}

func sc2Earth(fuel float64) *smd.Spacecraft {
	// Approximate planetary distance for distance reaching
	distance := smd.Earth.HelioOrbit(time.Now()).RNorm()
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := fuel
	ref2Earth := &smd.WaypointAction{Type: smd.REFEARTH, Cargo: nil}
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	// Uses the min *and* max values, since it only depends on the argument of periapsis.
	var i float64 = 31
	var Ω float64 = 330
	var ν float64 = 210
	hyper := smd.NewOrbitFromOE(a, e, i, Ω, 180, ν, smd.Mars)
	if withInjection {
		return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
			[]smd.Waypoint{
				smd.NewReachDistance(distance+smd.Earth.SOI, false, ref2Earth),
				smd.NewLoiter(time.Hour, nil),
				smd.NewToElliptical(nil),
				smd.NewOrbitTarget(*hyper, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL),
				smd.NewLoiter(7*24*time.Hour, nil),
			})
	}
	return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{
			smd.NewReachDistance(distance+smd.Earth.SOI, false, ref2Earth),
			smd.NewLoiter(time.Hour, nil),
		})
}
