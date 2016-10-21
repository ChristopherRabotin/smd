package main

import (
	"dynamics"
	"math"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	name := "IT1E"
	/* Building propagation */
	start := time.Now()                                   // Propagate starting now for ease.
	end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	// Create JSON output.
	CGOut(name, "Earth", start, end)
	sc := SpacecraftFromEarth(name)
	orbit := InitialEarthOrbit()
	astro := dynamics.NewAstro(sc, orbit, &start, &end, name)
	astro.Propagate()

	name = "IT1M"
	/* Building propagation */
	start = time.Now()                                   // Propagate starting now for ease.
	end = start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	// Create JSON output.
	CGOut(name, "Mars", start, end)
	astro = dynamics.NewAstro(SpacecraftFromMars(name), InitialMarsOrbit(), &start, &end, name)
	astro.Propagate()
}
