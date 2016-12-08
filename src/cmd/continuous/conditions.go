package main

import (
	"dynamics"
	"time"
)

// InitialEarthOrbit returns the initial orbit.
func InitialEarthOrbit() *dynamics.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	// SES-9 was delivered differently: http://spaceflight101.com/falcon-9-ses-9/ses-9-launch-success/
	a, e := dynamics.Radii2ae(39300+dynamics.Earth.Radius, 290+dynamics.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Earth)
}

// FromEarthWaypoints returns the waypoints.
func FromEarthWaypoints(destination dynamics.Orbit) []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	ref2Mars := &dynamics.WaypointAction{Type: dynamics.REFMARS, Cargo: nil}
	return []dynamics.Waypoint{
		// Loiter for 12 hours (eg. IOT)
		dynamics.NewLoiter(time.Duration(12)*time.Hour, nil),
		// Change the inclination by 12 degrees
		//dynamics.NewRelativeOrbitTarget(nil, []dynamics.RelativeOE{dynamics.RelativeOE{Law: dynamics.OptiΔiCL, Value: 12.0}}),
		// Leave Earth
		dynamics.NewOutwardSpiral(dynamics.Earth, ref2Sun),
		// Go straight to Mars destination
		dynamics.NewOrbitTarget(destination, ref2Mars),
		// Wait a week on arrival
		dynamics.NewLoiter(time.Duration(24*7)*time.Hour, nil)}
}

// InitialMarsOrbit returns the initial orbit.
func InitialMarsOrbit() *dynamics.Orbit {
	// Exomars TGO.
	a, e := dynamics.Radii2ae(44500+dynamics.Mars.Radius, 426+dynamics.Mars.Radius)
	i := 10.0
	ω := 1.0
	Ω := 1.0
	ν := 15.0
	return dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, dynamics.Mars)
}

// FromMarsWaypoints returns the waypoints.
func FromMarsWaypoints() []dynamics.Waypoint {
	ref2Sun := &dynamics.WaypointAction{Type: dynamics.REFSUN, Cargo: nil}
	return []dynamics.Waypoint{dynamics.NewOutwardSpiral(dynamics.Mars, ref2Sun)}
}
