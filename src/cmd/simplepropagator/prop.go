package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ChristopherRabotin/planetx-sim/dynamics"
)

func main() {
	log.Println("Plotting two orbits at the same time.")
	start := time.Now() // Propagate starting now for ease.
	end := start.Add(time.Duration(24) * time.Hour)
	var wg sync.WaitGroup
	// Eutelsat 117W (2016-038A)
	rP := 395.0 + dynamics.Earth.Radius
	rA := 62591.0 + dynamics.Earth.Radius
	a0, e0 := dynamics.Radii2ae(rA, rP)
	i0 := dynamics.Deg2rad(24.68)
	ω0 := dynamics.Deg2rad(10)
	Ω0 := dynamics.Deg2rad(5)
	ν0 := dynamics.Deg2rad(1)
	oEutel := dynamics.NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, &dynamics.Earth)
	log.Println(fmt.Sprintf("[Eutelsat] Body %s %s", dynamics.Earth.Name, oEutel))
	wg.Add(1)
	go func() {
		defer wg.Done()
		astro := dynamics.NewAstro(&dynamics.Spacecraft{Name: "Eutelsat", Mass: 1500}, oEutel, &start, &end, "../outputdata/propEutel")
		// Start propagation.
		log.Printf("[Eutel] starting propagation")
		astro.Propagate()
		log.Printf("[Eutel] propagation ended")
		fmt.Printf("[Eutel] Final orbital parameters %s\n", oEutel.String())
	}()
	// Define propagation parameters.

	// ISS
	a, e := dynamics.Radii2ae(409.5+dynamics.Earth.Radius, 400.2+dynamics.Earth.Radius)
	i1 := dynamics.Deg2rad(51.64)
	ω1 := dynamics.Deg2rad(10) // Made up
	Ω1 := dynamics.Deg2rad(5)  // Made up
	ν1 := dynamics.Deg2rad(1)  // I don't care about that guy.
	oISS := dynamics.NewOrbitFromOE(a, e, i1, ω1, Ω1, ν1, &dynamics.Earth)
	wg.Add(1)
	go func() {
		defer wg.Done()
		astro := dynamics.NewAstro(&dynamics.Spacecraft{Name: "test", Mass: 1500}, oISS, &start, &end, "../outputdata/propISS")
		// Start propagation.
		log.Printf("starting propagation")
		astro.Propagate()
		log.Printf("propagation ended")
	}()
	wg.Wait()
}
