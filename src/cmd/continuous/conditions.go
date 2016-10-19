package main

import (
	"dynamics"
	"time"
)

// InitialOrbit returns the initial orbit.
func InitialOrbit() *dynamics.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	a, e := dynamics.Radii2ae(350+dynamics.Earth.Radius, 250+dynamics.Earth.Radius)
	i := dynamics.Deg2rad(24.68)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, &dynamics.Earth)
}

// Waypoints returns the waypoints.
func Waypoints() []dynamics.Waypoint {
	return []dynamics.Waypoint{dynamics.NewOutwardSpiral(dynamics.Earth, nil),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
}
