package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	osc := smd.NewOrbitFromRV([]float64{-2436.45, -2436.45, 6891.037}, []float64{5.088611, -5.088611, 0}, smd.Earth)
	export := smd.ExportConfig{Filename: "hw0", Cosmo: false, AsCSV: true, Timestamp: false}
	ξ0 := osc.Getξ()
	prevV := osc.GetVNorm()
	export.CSVAppendHdr = func() string {
		return "energy,r,v,acc"
	}
	export.CSVAppend = func(st smd.MissionState) string {
		// Energy, |r|, |v|, |acc|
		acc := st.Orbit.GetVNorm() - prevV
		prevV = st.Orbit.GetVNorm()
		return fmt.Sprintf("%.15f,%.3f,%.6f,%.6f", st.Orbit.Getξ()-ξ0, st.Orbit.GetRNorm(), st.Orbit.GetVNorm(), acc)
	}
	start := time.Now().UTC()
	smd.NewMission(smd.NewEmptySC("hw", 0), osc, start, start.Add(osc.GetPeriod()*2), false, export).Propagate()
}
