package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ChristopherRabotin/planetx-sim/dynamics"
	"github.com/go-kit/kit/log"
)

func main() {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
	fmt.Println("Simple orbit propagator and visualizer.")
	// Eutelsat 117W (2016-038A)
	rP := 395.0
	rA := 62591.0
	a0 := rP + rA
	e0 := (rA - rP) / 2
	i0 := dynamics.Deg2rad(24.68)
	ω0 := dynamics.Deg2rad(10)
	Ω0 := dynamics.Deg2rad(5)
	ν0 := dynamics.Deg2rad(1)
	o := dynamics.NewOrbitFromOE(a0, e0, i0, ω0, Ω0, ν0, &dynamics.Earth)
	// Define propagation parameters.
	start, _ := time.Parse(time.RFC822, "01 Jan 15 10:00 UTC")
	end := start.Add(time.Duration(1) * time.Second)
	astro := dynamics.NewAstro(&dynamics.Spacecraft{Name: "test", Mass: 1500}, o, &start, &end, "../data/prop-")
	// Start propagation.
	logger.Log("starting propagation")
	astro.Propagate()
	logger.Log("propagation ended")
}
