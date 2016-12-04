package main

import (
	"dynamics"
	"time"
)

func main() {
	/* Simulate the research by Ruggiero et al. */

	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	iI := dynamics.Deg2rad(51.6)
	iT := dynamics.Deg2rad(46)
	initOrbit := dynamics.NewOrbitFromOE(350+dynamics.Earth.Radius, 0.01, iI, ω, Ω, ν, dynamics.Earth)
	targetOrbit := dynamics.NewOrbitFromOE(350+dynamics.Earth.Radius, 0.01, iT, ω, Ω, ν, dynamics.Earth)

	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	//thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	thrusters := []dynamics.Thruster{new(dynamics.PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	waypoints := []dynamics.Waypoint{dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil),
		dynamics.NewOrbitTarget(*targetOrbit, nil),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
	sc := dynamics.NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, []*dynamics.Cargo{}, waypoints)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.
	//end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Let's not have this last too long if it doesn't converge.

	sc.LogInfo()
	astro, wg := dynamics.NewAstro(sc, initOrbit, start, end, "Rug")
	astro.Propagate()

	wg.Wait() // Must wait or the output file does not have time to be written!
}
