package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

/* The goal of this example is to show the difference between a pure semi major axis increasing spiral
 * and a pure velocity increasing spiral, both in and out of a planet.
 */

func sc() *smd.Spacecraft {
	eps := smd.NewUnlimitedEPS()
	//thrusters := []smd.Thruster{&smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}, &smd.HPHET12k5{}}
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft("Spiral", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{}, []smd.Waypoint{smd.NewOutwardSpiral(smd.Earth, nil)})
}

func initOrbit() *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3)

	depart := time.Date(2015, 8, 30, 0, 0, 0, 0, time.UTC)
	endDT := time.Date(2016, 2, 27, 0, 0, 0, 0, time.UTC)
	name := "spiral-a2"
	astro := smd.NewMission(sc(), initOrbit(), depart, endDT, smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: true, Cosmo: true, Timestamp: false})
	astro.Propagate()

}

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}