package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	osc := smd.NewOrbitFromRV([]float64{-2436.45, -2436.45, 6891.037}, []float64{5.088611, -5.088611, 0}, smd.Earth)
	export := smd.ExportConfig{Filename: "hw0", Cosmo: false, AsCSV: true, Timestamp: false}
	ξ0 := osc.Energyξ()
	prevV := osc.VNorm()
	export.CSVAppendHdr = func() string {
		return "energy,r,v,acc"
	}
	export.CSVAppend = func(st smd.MissionState) string {
		// Energy, |r|, |v|, |acc|
		acc := st.Orbit.VNorm() - prevV
		prevV = st.Orbit.VNorm()
		return fmt.Sprintf("%.15f,%.3f,%.6f,%.6f", st.Orbit.Energyξ()-ξ0, st.Orbit.RNorm(), st.Orbit.VNorm(), acc)
	}
	start := time.Now().UTC()
	smd.NewMission(smd.NewEmptySC("hw", 0), osc, start, start.Add(osc.Period()*2), smd.GaussianVOP, smd.Perturbations{}, export).Propagate()
}
