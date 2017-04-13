package main

import (
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
)

/*
Simple propagation tool
*/
/*
X = -274096790.0 km
Y = -92859240.0 km
Z = -40199490.0 km
VX = 32.67 km/sec
VY = -8.94 km/sec
VZ = -3.88 km/sec
CR = 1.2
*/
func main() {
	//orbit := smd.NewOrbitFromRV([]float64{-274096790.0, -92859240.0, -40199490.0}, []float64{32.67, -8.94, -3.88}, smd.Earth)
	//-2.740967962303500  -0.928592250962256  -0.401995088201662   0.000000326707274  -0.000000089374725  -0.000000038789512
	orbit := smd.NewOrbitFromRV([]float64{-2.740967962303500e8, -0.928592250962256e8, -0.401995088201662e8}, []float64{32.6707274, -8.9374725, -3.8789512}, smd.Earth)
	sc := smd.NewEmptySC("Part2", 0)
	sc.Drag = 1.0
	startDT := julian.JDToTime(2456296.25)
	endDT := startDT.AddDate(0, 0, 50)
	perts := smd.Perturbations{PerturbingBody: &smd.Sun, Drag: true}
	smd.NewPreciseMission(sc, orbit, startDT, endDT, perts, 10*time.Second, false, smd.ExportConfig{AsCSV: false, Cosmo: true, Filename: "sprop-a"}).Propagate()
}
