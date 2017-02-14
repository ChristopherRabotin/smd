package main

import "github.com/ChristopherRabotin/smd"

// OutboundHyp returns the spacecraft.
func OutboundHyp(name string) *smd.Spacecraft {
	/* Building spacecraft */
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, OutboundWaypoints())
}

// InboundSpacecraft returns the spacecraft returning to Earth.
func InboundSpacecraft(name string, target smd.Orbit) *smd.Spacecraft {
	/* Building spacecraft */
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 2000.0 // Only 2/5 of the initial fuel.
	return smd.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, FromMarsWaypoints(target))
}
