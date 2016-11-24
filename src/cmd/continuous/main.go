package main

import (
	"dynamics"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3) // I'm running other stuff currently.
	// Propagate from Mars to heliocentric first.
	// Then use heliocentric velocity as the target velocity.
	/*
		name := "IM"
		/* Building propagation * /
		start := time.Now().UTC()                             // Propagate starting now for ease.
		end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
		astro, _ := dynamics.NewAstro(SpacecraftFromMars(name), InitialMarsOrbit(), start, end, name)
		astro.Propagate()
		fmt.Printf("Goal: r (km) = %.3f\tv (km/s) = %.3f\n", norm(astro.Orbit.R), norm(astro.Orbit.V))
		energyGoal := astro.Orbit.Energy()*/
	energyGoal := -287.1 // Computed previously.
	var wg sync.WaitGroup
	// Let's find the closest approach we can get to Mars by altering the ratio of when to slow down.
	for ratio := 0.6; ratio > 0.2; ratio -= 0.3 {
		wg.Add(1)
		name := fmt.Sprintf("IE%.1f", ratio)
		sc := SpacecraftFromEarth(name)
		sc.WayPoints = append(sc.WayPoints, dynamics.NewReachEnergy(energyGoal, ratio, nil))
		sc.WayPoints = append(sc.WayPoints, dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil))
		sc.LogInfo()
		go func() {
			defer wg.Done()
			astro, _ := dynamics.NewAstro(sc, InitialEarthOrbit(), start, end, name)
			astro.Propagate()
		}()
	}

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
