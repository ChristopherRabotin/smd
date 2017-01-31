package main

import (
	"time"

	"github.com/ChristopherRabotin/smd"
)

/*
Create a numerical simulation of a spacecraft in an orbit with a = 7000 km, e = 0.001, i = 30 degrees, Ω = 80 degrees, ω = 40 degrees,
and an initial true anomaly of ν = 0 degrees. Include only μ and J2 in the dynamics for the system.
Integrate the system for 24 hours. This is your reference trajectory.
Integrate a second trajectory by perturbing the initial state with a state deviation vector, δx = [1km, 0, 0, 0, 10m/s, 0].
Compare the state of the second trajectory with respect to the reference trajectory over the course of 24 hours.
Use the STM computed around the reference trajectory to perform a second propagation of δx.
*/
func main() {
	dt := time.Now().UTC()
	osc := smd.NewOrbitFromOE(7000, 0.001, 30, 80, 40, 0, smd.Earth)
	R, V := osc.RV()
	pert := smd.Perturbations{Jn: 2}
	mis := smd.NewMission(smd.NewEmptySC("hw10", 0), osc, dt, dt.Add(24*time.Hour), smd.Cartesian, pert, smd.ExportConfig{Filename: "hw1.0", Cosmo: true, AsCSV: true, Timestamp: false})
	mis.Propagate()

	// Second with initial error δx.
	R[0]++
	V[1] += 10e-3
	osc = smd.NewOrbitFromRV(R, V, smd.Earth)
	mis = smd.NewMission(smd.NewEmptySC("hw11", 0), osc, dt, dt.Add(24*time.Hour), smd.Cartesian, pert, smd.ExportConfig{Filename: "hw1.1", Cosmo: true, AsCSV: true, Timestamp: false})
	mis.Propagate()
}
