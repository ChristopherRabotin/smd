package main

import (
	"dynamics"
	"time"
)

func main() {
	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC)
	end := time.Date(2016, 10, 13, 0, 0, 0, 0, time.UTC)
	sc := dynamics.NewEmptySC("test", 100)
	// OEs
	//earth := dynamics.Earth.HelioOrbit(start)
	//mars := dynamics.Mars.HelioOrbit(start)
	//a := 0.5 * (earth.GetApoapsis() + mars.GetApoapsis())
	a := 1.5 * dynamics.Earth.Radius
	e := 1e-7
	i := 1e-7
	Ω := 90.0
	ω := 45.0
	ν := 20.5
	mss := dynamics.NewMission(sc, dynamics.NewOrbitFromOE(a, e, i, Ω, ω, ν, dynamics.Earth), start, end, false, dynamics.ExportConfig{Filename: "Inc", OE: true, Cosmo: true, Timestamp: false})
	mss.Propagate()
}
