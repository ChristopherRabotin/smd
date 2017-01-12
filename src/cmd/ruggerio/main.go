package main

import (
	"dynamics"
	"time"
)

func main() {
	/* Simulate the research by Ruggiero et al. */

	ω := 10.0 // Made up
	Ω := 5.0  // Made up
	ν := 1.0  // I don't care about that guy.

	initOrbit := dynamics.NewOrbitFromOE(350+dynamics.Earth.Radius, 0.01, 46, Ω, ω, ν, dynamics.Earth)
	targetOrbit := dynamics.NewOrbitFromOE(350+dynamics.Earth.Radius, 0.01, 46, 0.0, ω, ν, dynamics.Earth)

	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	//thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	thrusters := []dynamics.Thruster{new(dynamics.PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	waypoints := []dynamics.Waypoint{dynamics.NewOrbitTarget(*targetOrbit, nil, dynamics.Ruggerio)}
	sc := dynamics.NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, []*dynamics.Cargo{}, waypoints)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(54*24) * time.Hour)     // Propagate for 54 days.
	//end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Let's not have this last too long if it doesn't converge.

	sc.LogInfo()
	astro := dynamics.NewMission(sc, initOrbit, start, end, dynamics.ExportConfig{Filename: "Rugg", OE: true, Cosmo: false, Timestamp: false})
	astro.Propagate()

}
