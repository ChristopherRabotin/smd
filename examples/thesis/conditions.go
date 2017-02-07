package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

// InitialEarthOrbit returns the initial orbit.
func InitialEarthOrbit() *smd.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	// SES-9 was delivered differently: http://spaceflight101.com/falcon-9-ses-9/ses-9-launch-success/
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// FromEarthWaypoints returns the waypoints.
func FromEarthWaypoints(target smd.Orbit) []smd.Waypoint {
	fmt.Printf("[TARGET] %s\n", target)
	ref2Mars := &smd.WaypointAction{Type: smd.REFMARS, Cargo: nil}
	return []smd.Waypoint{
		// Leave Earth
		smd.NewOutwardSpiral(smd.Earth, nil),
		// Fix argument of periapsis and RAAN
		//smd.NewOrbitTarget(target, nil, smd.Naasz, smd.OptiΔΩCL, smd.OptiΔωCL),
		// Go straight to Mars destination
		smd.NewOrbitTarget(target, ref2Mars, smd.Naasz, smd.OptiΔaCL, smd.OptiΔeCL, smd.OptiΔiCL),
		// Wait a week on arrival
		smd.NewLoiter(time.Duration(3*24)*time.Hour, nil)}
}

// InitialMarsOrbit returns the initial orbit.
func InitialMarsOrbit() *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	i := 10.0
	ω := 1.0
	Ω := 1.0
	ν := 15.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

// FromMarsWaypoints returns the waypoints.
func FromMarsWaypoints() []smd.Waypoint {
	ref2Sun := &smd.WaypointAction{Type: smd.REFSUN, Cargo: nil}
	return []smd.Waypoint{smd.NewOutwardSpiral(smd.Mars, ref2Sun)}
}
