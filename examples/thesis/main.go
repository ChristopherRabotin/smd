package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	baseDepart := time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC)
	maxPropDT := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	estArrival := baseDepart.Add(time.Duration(400*24) * time.Hour)
	// Get Mars orbit at estimated arrival date.
	dest := smd.Mars.HelioOrbit(estArrival)
	R := dest.R()
	// Let's target that orbit, but offset by a factor of the SOI
	R[0] -= smd.Mars.SOI * 2
	target := smd.NewOrbitFromRV(R, dest.V(), smd.Sun)
	name := "SC"
	// Propagate until all waypoints are reached.
	sc := OutboundSpacecraft(name, *target)
	sc.LogInfo()
	astro := smd.NewMission(sc, InitialOrbit(), baseDepart, maxPropDT, smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{Filename: name, AsCSV: true, Cosmo: true, Timestamp: false})
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

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}
