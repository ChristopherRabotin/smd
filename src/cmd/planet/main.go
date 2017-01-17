package main

import (
	"dynamics"
	"fmt"
	"time"
)

func main() {
	end := time.Now().UTC().Add(time.Duration(2) * time.Hour)
	start := end.Add(time.Duration(-5*30.5*24) * time.Hour)
	sc := dynamics.NewEmptySC("inc", 100)
	obj := dynamics.Earth
	oI := obj.HelioOrbit(start)
	/*a := 20 * obj.Radius
	e := 1e-1
	i := 1e-1
	Ω := 1e-1 //90.0
	ω := 1e-1 //45.0
	ν := 20.5
	oI := dynamics.NewOrbitFromOE(a, e, i, Ω, ω, ν, obj)*/
	R, V := oI.GetRV()
	oV := dynamics.NewOrbitFromRV(R, V, dynamics.Sun)
	fmt.Printf("oI: %s\noV: %s\n", oI, oV)
	mss := dynamics.NewMission(sc, &oI, start, end, false, dynamics.ExportConfig{Filename: "Inc", OE: true, Cosmo: true, Timestamp: false})
	mss.Propagate()
}
