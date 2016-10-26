package main

import (
	"dynamics"
	"fmt"
	"math"
	"os"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	name := "IT1E"
	/* Building propagation */
	start := time.Now()                                   // Propagate starting now for ease.
	end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	sc := SpacecraftFromEarth(name)
	orbit := InitialEarthOrbit()
	astro := dynamics.NewAstro(sc, orbit, start, end, name)
	astro.Propagate()

	name = "IT1M"
	/* Building propagation */
	start = time.Now()                                   // Propagate starting now for ease.
	end = start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	astro = dynamics.NewAstro(SpacecraftFromMars(name), InitialMarsOrbit(), start, end, name)
	astro.Propagate()
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
