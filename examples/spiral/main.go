package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func sc() *smd.Spacecraft {
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft("Spiral", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{smd.NewToHyperbolic(nil)})
}

func initEarthOrbit(ν float64) *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// initMarsOrbit returns the initial orbit.
func initMarsOrbit(ν float64) *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	i := 10.0
	ω := 1.0
	Ω := 1.0
	//ν := 15.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

func main() {
	//name := "spiral-mars"
	depart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	initOrbit := initEarthOrbit(137)
	astro := smd.NewMission(sc(), initOrbit, depart, depart.Add(-1), smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: "spiral", AsCSV: false, Cosmo: true, Timestamp: false})
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
