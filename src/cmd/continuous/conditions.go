package main

import (
	"dynamics"
	"time"
)

// InitialEarthOrbit returns the initial orbit.
func InitialEarthOrbit() *dynamics.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	a, e := dynamics.Radii2ae(350+dynamics.Earth.Radius, 250+dynamics.Earth.Radius)
	i := dynamics.Deg2rad(24.68)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Earth)
}

// FromEarthWaypoints returns the waypoints.
func FromEarthWaypoints() []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	return []dynamics.Waypoint{dynamics.NewLoiter(time.Duration(24*2)*time.Hour, nil),
		dynamics.NewOutwardSpiral(dynamics.Earth, ref2Sun),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
}

// InitialMarsOrbit returns the initial orbit.
func InitialMarsOrbit() *dynamics.Orbit {
	a, e := dynamics.Radii2ae(44500+dynamics.Mars.Radius, 426+dynamics.Mars.Radius)
	i := dynamics.Deg2rad(10)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Mars)
}

// FromMarsWaypoints returns the waypoints.
func FromMarsWaypoints() []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	return []dynamics.Waypoint{dynamics.NewLoiter(time.Duration(24*2)*time.Hour, nil),
		dynamics.NewOutwardSpiral(dynamics.Mars, ref2Sun),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
}
