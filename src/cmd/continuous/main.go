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
	// Propagate from Mars to heliocentric first.
	// Then use heliocentric velocity as the target velocity.

	name := "IM"
	/* Building propagation */
	start := time.Now().UTC()                             // Propagate starting now for ease.
	end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	astro, _ := dynamics.NewAstro(SpacecraftFromMars(name), InitialMarsOrbit(), start, end, name)
	astro.Propagate()
	fmt.Printf("Goal: r (km) = %.3f\tv (km/s) = %.3f\n", norm(astro.Orbit.R), norm(astro.Orbit.V))

	name = "IE"
	sc := SpacecraftFromEarth(name)
	sc.WayPoints = append(sc.WayPoints, dynamics.NewReachEnergy(astro.Orbit.Energy(), 0.5, nil))
	sc.WayPoints = append(sc.WayPoints, dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil))
	sc.LogInfo()
	orbit := InitialEarthOrbit()
	astro, wg := dynamics.NewAstro(sc, orbit, start, end, name)
	astro.Propagate()

	// Wait for the streams to finish writing.
	wg.Wait()
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
