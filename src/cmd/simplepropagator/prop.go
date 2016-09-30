package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ChristopherRabotin/planetx-sim/dynamics"
)

func main() {
	log.Println("Simple orbit propagator and visualizer.")
	// Eutelsat 117W (2016-038A)
	rP := 395.0
	rA := 62591.0
	a0 := (rP + rA) / 2
	e0 := (rA - rP) / (rA + rP)
	i0 := dynamics.Deg2rad(24.68)
	ω0 := dynamics.Deg2rad(10)
	Ω0 := dynamics.Deg2rad(5)
	ν0 := dynamics.Deg2rad(1)
	log.Println(fmt.Sprintf("[Orbit] Body %s a=%0.5f e=%0.5f i=%0.5f ω=%0.5f Ω=%0.5f ν=%0.5f", dynamics.Earth.Name, a0, e0, i0, ω0, Ω0, ν0))
	o := dynamics.NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, &dynamics.Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(1) * time.Second)
	astro := dynamics.NewAstro(&dynamics.Spacecraft{Name: "test", Mass: 1500}, o, &start, &end, "../outputdata/prop")
	// Start propagation.
	log.Printf("starting propagation")
	astro.Propagate()
	log.Printf("propagation ended")
}
