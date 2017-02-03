package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3)

	/* Propagate Mars only once. Then perform dichotomy to get to Mars. */

	baseDepart := time.Date(2015, 8, 30, 0, 0, 0, 0, time.UTC)
	initTimeStep := time.Duration(4*7*24) * time.Hour
	estArrival := time.Date(2016, 8, 24, 0, 0, 0, 0, time.UTC)

	marsStartDT := estArrival.Add(-time.Duration(2*31*24) * time.Hour)
	scMars := SpacecraftFromMars("IM")
	scMars.LogInfo()
	// Propagate the Mars orbit until it is heliocentric. We will target this because it means from there, we can return back into Mars SOI.
	astroM := smd.NewMission(scMars, InitialMarsOrbit(), marsStartDT, marsStartDT.Add(time.Duration(-1)*time.Hour), smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: "IM", AsCSV: false, Cosmo: false, Timestamp: false})
	astroM.Propagate()

	target := astroM.Orbit

	// Start dichotomy
	iter := 0
	arrivedEarly := 1 // if arrived early, then set this to 1 (to leave later).
	timeStep := initTimeStep
	for timeStep > time.Duration(1*24)*time.Hour {
		actualStart := baseDepart
		if iter > 0 {
			timeStep = time.Duration(arrivedEarly) * timeStep / time.Duration(iter)
			actualStart = actualStart.Add(timeStep)
		}
		name := fmt.Sprintf("IE-%d%1d%1d%1d", actualStart.Year(), actualStart.Month(), actualStart.Day(), actualStart.Hour())
		fmt.Printf("===== %s (iter=%d early=%d) =====\n", actualStart, iter, arrivedEarly)
		sc := SpacecraftFromEarth(name, *target)
		sc.LogInfo()
		// Only propagate til a bit after the estimated arrival date.
		maxDT := estArrival.Add(time.Duration(3*31*24) * time.Hour)
		astro := smd.NewMission(sc, InitialEarthOrbit(), actualStart, maxDT, smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: true, Cosmo: true, Timestamp: false})
		astro.Propagate()
		// Determine whether we arrived before or after Mars
		marsR0 := smd.Mars.HelioOrbit(astro.CurrentDT).R()
		marsR1 := smd.Mars.HelioOrbit(astro.CurrentDT.Add(time.Hour)).R()
		scR := astro.Orbit.R()
		diff0 := []float64{0, 0, 0}
		diff1 := []float64{0, 0, 0}
		for i := 0; i < 3; i++ {
			diff0[i] = marsR0[i] - scR[i]
			diff1[i] = marsR1[i] - scR[i]
		}
		if norm(diff0) > norm(diff1) {
			// This means the distance has *decreased*, therefore Mars is getting closer, so effectively,
			// we have to leave slightly later to hopefully catch Mars.
			arrivedEarly = 1
		} else {
			// We are late and Mars is leaving. Need to leave earlier to catch Mars.
			arrivedEarly = -1
		}
		iter++
	}

}

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}
