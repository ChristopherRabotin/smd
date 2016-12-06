package main

import "dynamics"

// SpacecraftFromEarth returns the spacecraft.
func SpacecraftFromEarth(name string, destination dynamics.Orbit) *dynamics.Spacecraft {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	//thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	thrusters := []dynamics.Thruster{dynamics.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return dynamics.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, []*dynamics.Cargo{}, FromEarthWaypoints(destination))
}

// SpacecraftFromMars returns the spacecraft.
func SpacecraftFromMars(name string) *dynamics.Spacecraft {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	//thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	thrusters := []dynamics.Thruster{dynamics.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return dynamics.NewSpacecraft(name, dryMass, fuelMass, eps, thrusters, []*dynamics.Cargo{}, FromMarsWaypoints())
}
