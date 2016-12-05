package main

import (
	"dynamics"
	"time"
)

func main() {
	/* Simulate the research by Ruggiero et al. */

	config := dynamics.ExportConfig{Filename: "Rug", Cosmo: true, OE: true, Timestamp: false}

	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)
	ν := dynamics.Deg2rad(1) // I don't care about that guy.
	i := dynamics.Deg2rad(7)
	i1 := dynamics.Deg2rad(0)
	e := dynamics.Deg2rad(0.7283)
	e1 := 0.0
	a := 24386.0
	a1 := 42164.0
	initOrbit := dynamics.NewOrbitFromOE(a, e, i, Ω, ω, ν, dynamics.Earth)
	targetOrbit := dynamics.NewOrbitFromOE(a1, e1, i1, Ω, ω, ν, dynamics.Earth)

	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{new(dynamics.PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	waypoints := []dynamics.Waypoint{dynamics.NewLoiter(time.Duration(1)*time.Hour, nil),
		dynamics.NewOrbitTarget(*targetOrbit, nil),
		dynamics.NewLoiter(time.Duration(1)*time.Hour, nil)}
	sc := dynamics.NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, []*dynamics.Cargo{}, waypoints)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.

	sc.LogInfo()
	astro, wg := dynamics.NewAstro(sc, initOrbit, start, end, config)
	astro.Propagate()

	wg.Wait() // Must wait or the output file does not have time to be written!
}
