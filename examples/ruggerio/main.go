package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	/* Building spacecraft */
	eps := smd.NewUnlimitedEPS()
	//thrusters := []smd.Thruster{&smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}}
	thrusters := []smd.EPThruster{new(smd.PPS1350)}
	dryMass := 300.0
	fuelMass := 67.0
	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	end := start.Add(time.Duration(60*24) * time.Hour)     // Propagate for 54 days.
	//end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Let's not have this last too long if it doesn't converge.

	var results string

	// See if any parameter leads to a substantial change of the inclination using the laws as defined in the paper.
	for Ω := 0.0; Ω < 360; Ω += 10.0 {
		for ω := 0.0; ω < 360; ω += 10.0 {
			for ν := 0.0; ν < 360; ν += 10.0 {
				initOrbit := smd.NewOrbitFromOE(350+smd.Earth.Radius, 0.01, 46, Ω, ω, ν, smd.Earth)
				targetOrbit := smd.NewOrbitFromOE(350+smd.Earth.Radius, 0.01, 51.6, Ω, ω, ν, smd.Earth)

				waypoints := []smd.Waypoint{smd.NewOrbitTarget(*targetOrbit, nil, smd.Ruggerio, smd.OptiΔiCL)}
				sc := smd.NewSpacecraft("Rug", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, waypoints)

				sc.LogInfo()
				astro := smd.NewMission(sc, initOrbit, start, end, false, smd.ExportConfig{Filename: "Rugg", OE: true, Cosmo: false, Timestamp: false})
				astro.Propagate()
				results += fmt.Sprintf("%s\tΩ=%f\tω=%f\tν=%f\n", initOrbit, Ω, ω, ν)
			}
		}
	}

	fmt.Printf(results)
}
