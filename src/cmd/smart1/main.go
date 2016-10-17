package main

import (
	"dynamics"
	"log"
	"math"
	"time"
)

func main() {
	log.Println("SMART-1")

	/* Building waypoints */
	outSpiral := dynamics.NewWaypoint(func(position *dynamics.Orbit) bool {
		return false // Always boosting
	})
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.PPS1350{}}
	waypoints := []*dynamics.Waypoint{outSpiral}
	dryMass := 287.0
	fuelMass := 82.0
	sc := &dynamics.Spacecraft{Name: "SMART-1", DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: waypoints}

	/* Building propagation */
	start := time.Now() // Propagate starting now for ease of visualization.
	end := start.Add(time.Duration(24*30) * time.Hour)
	//end := start.Add(time.Duration(2) * time.Hour)
	// Ariane 5 delivered it in GTO.
	a, e := dynamics.Radii2ae(42223, 7035)
	i := dynamics.Deg2rad(6.9)
	ω := dynamics.Deg2rad(90) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(90) // I don't care about that guy.
	orbit := dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, &dynamics.Earth)
	initV := math.Sqrt(orbit.V[0]*orbit.V[0] + orbit.V[1]*orbit.V[1] + orbit.V[2]*orbit.V[2])
	astro := dynamics.NewAstro(sc, orbit, &start, &end, "../outputdata/propSMART1")
	// Start propagation.
	log.Printf("Depart from: %s\n", orbit.String())
	astro.Propagate()
	log.Printf("Arrived at: %s\n", orbit.String())
	finalV := math.Sqrt(orbit.V[0]*orbit.V[0] + orbit.V[1]*orbit.V[1] + orbit.V[2]*orbit.V[2])
	log.Printf("Total deltaV = %f km/s (init = %f km/s, final = %f)", finalV-initV, initV, finalV)
}
