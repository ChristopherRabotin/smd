package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

// InitialOrbit returns the initial orbit.
func InitialOrbit() *smd.Orbit {
	// Falcon 9 delivers at 24.68 350x250km.
	// SES-9 was delivered differently: http://spaceflight101.com/falcon-9-ses-9/ses-9-launch-success/
	/*a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)*/
	// This is the orbit we get when launched on a hyperbolic trajectory.
	return smd.NewOrbitFromRV([]float64{1.4400176291063553e+08, -3.9243962075166054e+07, -1.6889461230736043e+07}, []float64{7.615139993500486, 26.125876826086497, 11.364911015802056}, smd.Sun)
}

// OutboundWaypoints returns the waypoints for the outbound spacecraft.
func OutboundWaypoints(target smd.Orbit) []smd.Waypoint {
	fmt.Printf("[TARGET] %s\n", target)
	//ref2Sun := &smd.WaypointAction{Type: smd.REFSUN, Cargo: nil}
	ref2Mars := &smd.WaypointAction{Type: smd.REFMARS, Cargo: nil}
	return []smd.Waypoint{
		// Leave Earth
		//smd.NewToHyperbolic(ref2Sun),
		// Go straight to Mars destination
		smd.NewOrbitTarget(target, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔeCL, smd.OptiΔiCL),
		// Now attempt to fix everything
		smd.NewOrbitTarget(target, ref2Mars, smd.Naasz),
		// Wait for the ref2Mars to trigger... ?
		smd.NewLoiter(time.Duration(1)*time.Minute, nil),
		// Make orbit Elliptical
		smd.NewToElliptical(nil),
		// Wait a week on arrival
		smd.NewLoiter(time.Duration(7*24)*time.Hour, nil)}
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
	return []smd.Waypoint{smd.NewToHyperbolic(ref2Sun)}
}
