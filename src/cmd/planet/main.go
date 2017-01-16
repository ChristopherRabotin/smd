package main

import (
	"dynamics"
	"time"
)

func main() {
	start := time.Now().UTC()
	end := start.Add(time.Duration(5*30.5*24) * time.Hour)
	sc := dynamics.NewEmptySC("test", 100)
	// OEs
	//earth := dynamics.Earth.HelioOrbit(start)
	//mars := dynamics.Mars.HelioOrbit(start)
	//a := 0.5 * (earth.GetApoapsis() + mars.GetApoapsis())
	a := 1.5 * dynamics.Mars.Radius
	e := 1e-7
	i := 1e-7
	Ω := 90.0
	ω := 45.0
	ν := 20.5
	mss := dynamics.NewMission(sc, dynamics.NewOrbitFromOE(a, e, i, Ω, ω, ν, dynamics.Mars), start, end, false, dynamics.ExportConfig{Filename: "Inc", OE: true, Cosmo: true, Timestamp: false})
	mss.Propagate()
}
