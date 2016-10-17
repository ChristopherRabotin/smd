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
	log.Println("Continous propagation")

	/* Building waypoints */
	outSpiral := dynamics.NewWaypoint(func(position *dynamics.Orbit) bool {
		return norm(position.R) >= dynamics.Earth.SOI
	})
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	//thrusters := []dynamics.Thruster{&dynamics.PPS1350{}, &dynamics.PPS1350{}}
	waypoints := []*dynamics.Waypoint{outSpiral}
	dryMass := 1000.0
	fuelMass := 500.0
	sc := &dynamics.Spacecraft{Name: "Continuous Prop test", DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: waypoints}

	/* Building propagation */
	start := time.Now() // Propagate starting now for ease.
	end := start.Add(time.Duration(24*30.5) * time.Hour)
	// Falcon 9 delivers at 24.68 350x250km.
	a, e := dynamics.Radii2ae(350+dynamics.Earth.Radius, 250+dynamics.Earth.Radius)
	/*a := 400 + dynamics.Earth.Radius
	e := 1e-2*/
	i := dynamics.Deg2rad(24.68)
	//i := dynamics.Deg2rad(0)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	orbit := dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, &dynamics.Earth)
	astro := dynamics.NewAstro(sc, orbit, &start, &end, "../outputdata/propCont")
	// Start propagation.
	log.Printf("Depart from: %s\n", orbit.String())
	astro.Propagate()
	log.Printf("Arrived at: %s\n", orbit.String())
}
