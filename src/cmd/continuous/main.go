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
	end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.
	name := "IE"
	sc := SpacecraftFromEarth(name)
	sc.LogInfo()
	astro, wg := dynamics.NewAstro(sc, InitialEarthOrbit(), start, end, name)
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
