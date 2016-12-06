package main

import (
	"dynamics"
	"time"
)

// InitialEarthOrbit returns the initial orbit.
func InitialEarthOrbit() *dynamics.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	// SES-9 was delivered differently: http://spaceflight101.com/falcon-9-ses-9/ses-9-launch-success/
	/*a, e := dynamics.Radii2ae(39300+dynamics.Earth.Radius, 290+dynamics.Earth.Radius)
	i := dynamics.Deg2rad(28.0)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.*/
	// From the last step before crash.
	a := 187176.235
	e := 0.610
	i := 10.000
	ω := 27.195
	Ω := 1.000
	ν := 133.637
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Earth)
}

// FromEarthWaypoints returns the waypoints.
func FromEarthWaypoints(destination dynamics.Orbit) []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	ref2Mars := &dynamics.WaypointAction{Type: dynamics.REFMARS, Cargo: nil}
	return []dynamics.Waypoint{ //dynamics.NewLoiter(time.Duration(24*2)*time.Hour, nil),
		dynamics.NewOutwardSpiral(dynamics.Earth, ref2Sun),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil),
		dynamics.NewOrbitTarget(destination, ref2Mars),
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
}

// InitialMarsOrbit returns the initial orbit.
func InitialMarsOrbit() *dynamics.Orbit {
	// Exomars TGO.
	a, e := dynamics.Radii2ae(44500+dynamics.Mars.Radius, 426+dynamics.Mars.Radius)
	i := 10.0
	ω := 1.0 // Made up
	Ω := 1.0 // Made up
	//ν := dynamics.Deg2rad(270) // I don't care about that guy.
	ν := 15.0
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Mars)
}

// FromMarsWaypoints returns the waypoints.
func FromMarsWaypoints() []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	return []dynamics.Waypoint{dynamics.NewLoiter(time.Duration(24*2)*time.Hour, nil),
		dynamics.NewOutwardSpiral(dynamics.Mars, ref2Sun)}
	// We don't loiter at the end because we want specifically the transition point.
}
