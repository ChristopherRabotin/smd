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

	//start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	start := time.Date(2015, 7, 1, 0, 0, 0, 0, time.UTC) // ExoMars launch date.
	estArrival := time.Date(2017, 7, 24, 0, 0, 0, 0, time.UTC)

	/*
		Algo for TOF targeting:
		1. Propagate from random date
		2. Check when hit SOI
		3. Compute duration needed to reach that point
		4. From mean anomaly, compute the time needed for Mars to reach that point (time = DT)
		4a. This leads to knowing how much in advance we are.
		5. Repeat full propagation (including Mars departure) with that DT difference.
	*/

	/* Let's propagate out of Mars at a guessed date of 7 months after launch date from Earth.
	Note that we only output the CSV because we don't need to visualize this.
	*/
	// *WITH THE CURRENT DATE AND TIME*, it takes one month and five days to leave the SOI. So let's propagate only for that time.
	marsEndDT := estArrival.Add(time.Duration(31*24) * time.Hour)
	scMars := SpacecraftFromMars("IM")
	scMars.LogInfo()
	astroM := smd.NewMission(scMars, InitialMarsOrbit(), estArrival, marsEndDT, smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: "IM", AsCSV: false, Cosmo: false, Timestamp: false})
	astroM.Propagate()
	// Convert the position to heliocentric.
	astroM.Orbit.ToXCentric(smd.Sun, astroM.CurrentDT)
	target := astroM.Orbit

	for incr := 0; incr < 9; incr++ {
		actualStart := start.Add(time.Duration(incr*15*24) * time.Hour) // Adding two week periods
		name := fmt.Sprintf("IE-%d%1d%1d", actualStart.Year(), actualStart.Month(), actualStart.Day())
		fmt.Printf("===== %s (%s) =====\n", actualStart, name)
		sc := SpacecraftFromEarth(name, *target)
		sc.LogInfo()
		// Don't propagate too long, it should only take about 8 anyway.
		maxDT := actualStart.Add(time.Duration(12*31*24) * time.Hour)
		astro := smd.NewMission(sc, InitialEarthOrbit(), actualStart, maxDT, smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: true, Cosmo: true, Timestamp: false})
		astro.Propagate()
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
