package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	end := time.Now().UTC().Add(time.Duration(2) * time.Hour)
	start := end.Add(time.Duration(-2*30.5*24) * time.Hour)
	sc := smd.NewEmptySC("inc", 100)
	obj := smd.Mars
	oI := obj.HelioOrbit(start)
	/*a := 20 * obj.Radius
	e := 1e-1
	i := 1e-1
	Ω := 1e-1 //90.0
	ω := 1e-1 //45.0
	ν := 20.5
	oI := smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, obj)*/
	R, V := oI.RV()
	oV := smd.NewOrbitFromRV(R, V, smd.Sun)
	fmt.Printf("oI: %s\noV: %s\n", oI, oV)
	mss := smd.NewMission(sc, &oI, start, end, smd.GaussianVOP, false, smd.ExportConfig{Filename: "Inc", AsCSV: true, Cosmo: true, Timestamp: false})
	mss.Propagate()
}
