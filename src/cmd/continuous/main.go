package main

import (
	"dynamics"
	"log"
	"math"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	name := "IT1"

	/* Building propagation */
	start := time.Now()                                   // Propagate starting now for ease.
	end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	// Create JSON output.
	CGOut(name, start, end)
	sc := Spacecraft(name)
	orbit := InitialOrbit()
	astro := dynamics.NewAstro(sc, orbit, &start, &end, name)
	// Start propagation.
	log.Printf("Depart from: %s\n", orbit.String())
	astro.Propagate()
	log.Printf("Arrived at: %s\n", orbit.String())
}
