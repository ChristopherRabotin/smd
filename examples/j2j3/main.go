package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {

	startDT := time.Now()
	endDT := startDT.Add(24 * time.Hour)

	deltaX := []float64{}
	DThrs := []float64{}
	// Define the special export functions
	export0 := smd.ExportConfig{Filename: "j2", Cosmo: false, AsCSV: true, Timestamp: false}
	export0.CSVAppendHdr = func() string {
		return ""
	}
	export0.CSVAppend = func(state smd.MissionState) string {
		deltaX = append(deltaX, state.Orbit.R()[0])
		DThrs = append(DThrs, state.DT.Sub(startDT).Hours())
		return ""
	}

	// Generate the perturbed orbit
	smd.NewMission(smd.NewEmptySC("LEO", 0), smd.NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, smd.Earth), startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 2}, export0).Propagate()

	ex1 := 0
	export1 := smd.ExportConfig{Filename: "j3", Cosmo: false, AsCSV: true, Timestamp: false}
	export1.CSVAppendHdr = func() string {
		return ""
	}
	export1.CSVAppend = func(state smd.MissionState) string {
		deltaX[ex1] -= state.Orbit.R()[0]
		ex1++
		return ""
	}
	smd.NewMission(smd.NewEmptySC("LEO", 0), smd.NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, smd.Earth), startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 3}, export1).Propagate()

	// Export deltas
	f, err := os.Create("./deltaX.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("hours,deltaX\n")
	for i, delta := range deltaX {
		csv := fmt.Sprintf("%f,%f\n", DThrs[i], delta)
		if _, err := f.WriteString(csv); err != nil {
			panic(err)
		}
	}
}
