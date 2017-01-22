package main

import (
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	runtime.GOMAXPROCS(3)
	sc := smd.NewEmptySC("IMD", 0) // Massless spacecraft! =D
	start := time.Date(2017, 1, 2, 3, 4, 5, 6, time.UTC)
	end := start.Add(time.Duration(-1) * time.Second)
	sc.LogInfo()
	initEarthOrbit := smd.NewOrbitFromOE(smd.Earth.Radius+400, 0, 0, 0, 0, 0, smd.Earth)
	soiEarthOrbit := smd.NewOrbitFromOE(smd.Earth.SOI, 0, 0, 0, 0, 0, smd.Earth)
	sc.WayPoints = []smd.Waypoint{smd.NewHohmannTransfer(*soiEarthOrbit, nil)}
	mss := smd.NewMission(sc, initEarthOrbit, start, end, false, smd.ExportConfig{})
	mss.Propagate()
}
