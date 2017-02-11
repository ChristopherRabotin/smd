package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	baseDepart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	// Est arrival is best according to lambert solver
	estArrival := time.Date(2019, 03, 23, 0, 0, 0, 0, time.UTC)
	maxPropDT := estArrival.Add(time.Duration(24*31*24) * time.Hour)

	// Find target hyperbola
	hypDepart := estArrival.Add(time.Duration(-6*31*24) * time.Hour)
	hypsc := OutboundHyp("hypSC")
	hyp := smd.NewMission(hypsc, finalGTO(), hypDepart, hypDepart.Add(-1), smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{})
	hyp.Propagate()

	// Now let's grab the final hyperbolic orbit as the target.
	inb := smd.NewPreciseMission(InboundSpacecraft("inbSC", *hyp.Orbit), InitialMarsOrbit(), baseDepart, maxPropDT, smd.Cartesian, smd.Perturbations{}, time.Minute, smd.ExportConfig{AsCSV: false, Cosmo: true, Filename: "inb"})
	inb.Propagate()
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
