package main

import "dynamics"

// Spacecraft returns the spacecraft.
func Spacecraft(name string) *dynamics.Spacecraft {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	//thrusters := []dynamics.Thruster{&dynamics.PPS1350{}, &dynamics.PPS1350
	dryMass := 1000.0
	fuelMass := 500.0
	return &dynamics.Spacecraft{Name: name, DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: Waypoints()}
}
