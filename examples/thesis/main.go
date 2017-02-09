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
		// Get Mars orbit at estimated arrival date.
		dest := smd.Mars.HelioOrbit(estArrival)
		name := "SC"
		// Propagate until all waypoints are reached.
		sc := OutboundSpacecraft(name, dest)
		sc.LogInfo()
		astro := smd.NewMission(sc, InitialOrbit(), baseDepart, maxPropDT, smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: true, Cosmo: true, Timestamp: false})
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
