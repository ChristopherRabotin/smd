package main

import "dynamics"

// SpacecraftFromEarth returns the spacecraft.
func SpacecraftFromEarth(name string) *dynamics.Spacecraft {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	dryMass := 1000.0
	fuelMass := 500.0
	return &dynamics.Spacecraft{Name: name, DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: FromEarthWaypoints(), FuncQ: make([]func(), 5)}
}

// SpacecraftFromMars returns the spacecraft.
func SpacecraftFromMars(name string) *dynamics.Spacecraft {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	dryMass := 1000.0
	fuelMass := 500.0
	return &dynamics.Spacecraft{Name: name, DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: FromMarsWaypoints(), FuncQ: make([]func(), 5)}
}
