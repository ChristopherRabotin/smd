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
	return smd.NewSpacecraft("Spiral", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{smd.NewToHyperbolic(nil), smd.NewToElliptical(nil), smd.NewLoiter(time.Duration(32000)*time.Hour, nil)})
}

func initEarthOrbit() *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// initMarsOrbit returns the initial orbit.
func initMarsOrbit() *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	i := 10.0
	ω := 1.0
	Ω := 1.0
	ν := 15.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

func main() {
	//depart := time.Date(2015, 8, 30, 0, 0, 0, 0, time.UTC)
	depart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	endDT := depart.Add(-1)
	name := "spiral-mars"
	astro := smd.NewMission(sc(), initMarsOrbit(), depart, endDT, smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: false, Cosmo: true, Timestamp: false})
	astro.Propagate()
}

func init() {
	runtime.GOMAXPROCS(3)
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}
