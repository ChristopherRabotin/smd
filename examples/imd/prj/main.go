package main

import (
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	//startDT := time.Date(2048, 10, 18, 03, 06, 9, 260, time.UTC)
	startDT := time.Date(2050, 03, 02, 03, 06, 9, 260, time.UTC).Add(-time.Duration(473*24) * time.Hour)
	endDT := startDT.AddDate(2, 0, 0)
	//position := []float64{-1646594, -3671650, 1438363}
	//velocity := []float64{6.38634452, 0.965306667, 0.7707777599}
	position := []float64{-4347331, -1599287, -1627769.95}
	velocity := []float64{-1.264929, -0.283112, -0.114028}

	orbit := smd.NewOrbitFromRV(position, velocity, smd.Neptune)

	//wp := []smd.Waypoint{smd.NewToElliptical(nil)}
	wp := []smd.Waypoint{}
	sc := smd.NewSpacecraft("CRA", 904, 3000, smd.NewUnlimitedEPS(), []smd.EPThruster{new(smd.BHT8000), new(smd.BHT8000), new(smd.BHT8000), new(smd.BHT8000)}, false, []*smd.Cargo{}, wp)

	smd.NewPreciseMission(sc, orbit, startDT, endDT, smd.Perturbations{}, 30*time.Second, false, smd.ExportConfig{Cosmo: true, Filename: "CRA"}).Propagate()
}
