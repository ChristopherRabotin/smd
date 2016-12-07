package main

import (
	"dynamics"
	"fmt"
	"math"
	"os"
	"runtime"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3) // I'm running other stuff currently.

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	//end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.
	end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Let's not have this last too long if it doesn't converge.

	/* Let's propagate out of Mars at a guessed date of 7 months after launch date from Earth.
	Note that we only output the CSV because we don't need to visualize this.
	*/

	scMars := SpacecraftFromMars("IM")
	scMars.LogInfo()
	astroM, wg := dynamics.NewAstro(scMars, InitialMarsOrbit(), start.Add(time.Duration(7*31*24)*time.Hour), start, dynamics.ExportConfig{Filename: "IM", OE: false, Cosmo: false, Timestamp: false})
	astroM.Propagate()
	wg.Wait() // Wait for files to finish writing before we continue.

	sc := SpacecraftFromEarth("IE", *astroM.Orbit)
	sc.LogInfo()
	astro, wg := dynamics.NewAstro(sc, InitialEarthOrbit(), start, end, dynamics.ExportConfig{Filename: "IE", OE: true, Cosmo: true, Timestamp: false})
	astro.Propagate()

	wg.Wait() // Must wait or the output file does not have time to be written!
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
