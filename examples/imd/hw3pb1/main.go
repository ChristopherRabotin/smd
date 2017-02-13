package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
)

func main() {
	jde := 2454085.5
	marsOrbit := smd.Mars.HelioOrbitAtJD(jde)
	marsR, marsV := marsOrbit.RV()
	jupiterDT := julian.JDToTime(jde).Add(time.Duration(830*24) * time.Hour)
	jupiterOrbit := smd.Jupiter.HelioOrbit(jupiterDT)
	jupiterR, jupiterV := jupiterOrbit.RV()
	fmt.Printf("==== Mars @%s ====\nR=%+v km\tV=%+v km/s\n", julian.JDToTime(jde), marsR, marsV)
	fmt.Printf("==== Jupiter @%s ====\nR=%+v km\tV=%+v km/s\n", jupiterDT, jupiterR, jupiterV)
}
