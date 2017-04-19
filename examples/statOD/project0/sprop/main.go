package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
)

func main() {
	orbit := smd.NewOrbitFromRV([]float64{-2.740967962303500e8, -0.928592250962256e8, -0.401995088201662e8}, []float64{32.6707274, -8.9374725, -3.8789512}, smd.Earth)
	sc := smd.NewEmptySC("Part2", 0)
	sc.Drag = 1.0
	startDT := julian.JDToTime(2456296.25)
	endDT := julian.JDToTime(2456346.2539)
	fmt.Printf("%+v\n", smd.MxV33(smd.R1(smd.Deg2rad(-23.4393)), smd.Earth.HelioOrbit(endDT).R()))
	if true {
		panic("stop")
	}
	perts := smd.Perturbations{PerturbingBody: &smd.Sun, Drag: true}
	smd.NewPreciseMission(sc, orbit, startDT, endDT, perts, 10*time.Second, false, smd.ExportConfig{AsCSV: false, Cosmo: true, Filename: "sprop-a"}).Propagate()
}
