package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

const (
	outbound = true // Set to False to simulate the inbound trajectory
)

func main() {
	if outbound {
		baseDepart := time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC)
		maxPropDT := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
		// The estimated arrival was computed from the minimum of a Lambert solver.
		estArrival := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
		//estArrival := time.Date(2019, 5, 20, 0, 0, 0, 0, time.UTC)
		// Compute hyperbolic exit of Mars from TGO injection orbit
		inb := smd.NewMission(InboundSpacecraft("IM"), InitialMarsOrbit(), estArrival.Add(-70*24*time.Hour), estArrival.Add(-71*24*time.Hour), smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: "IM", AsCSV: false, Cosmo: false, Timestamp: false})
		inb.Propagate()
		//target := *smd.NewOrbitFromRV([]float64{-2.3170735800800942e+07, 2.1403532352397686e+08, 9.837920496548975e+07}, []float64{-23.18157341593323, -2.5740622275206446, -0.9789467049939157}, smd.Sun)
		target := *inb.Orbit
		name := "SC"
		// Propagate until all waypoints are reached.
		sc := OutboundSpacecraft(name, target)
		sc.LogInfo()
		astro := smd.NewPreciseMission(sc, InitialOrbit(), baseDepart, maxPropDT, smd.Cartesian, smd.Perturbations{}, time.Minute, smd.ExportConfig{Filename: name, AsCSV: false, Cosmo: true, Timestamp: false})
		astro.Propagate()
	} else {
		// Return trajectory from Mars
		/*baseDepart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
		maxPropDT := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
		estArrival := time.Date(2019, 03, 23, 0, 0, 0, 0, time.UTC)*/
	}
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

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}
