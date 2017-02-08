package main

import "github.com/ChristopherRabotin/smd"

// OutboundSpacecraft returns the spacecraft.
func OutboundSpacecraft(name string, target smd.Orbit) *smd.Spacecraft {
	/* Building spacecraft */
	eps := smd.NewUnlimitedEPS()
	//thrusters := []smd.Thruster{&smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}}
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, OutboundWaypoints(target))
}

// SpacecraftFromMars returns the spacecraft.
func SpacecraftFromMars(name string) *smd.Spacecraft {
	/* Building spacecraft */
	eps := smd.NewUnlimitedEPS()
	//thrusters := []smd.Thruster{&smd.HPHET12k5{}, &smd.HPHET12k5{}}
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, FromMarsWaypoints())
}
