package main

import (
	"fmt"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	for i, a := range []float64{290 + smd.Earth.Radius, 39300 + smd.Earth.Radius} {
		initOrbit := smd.NewOrbitFromOE(a, 0, 0.0, 0, 0, 180, smd.Earth)
		tgtOrbit := smd.NewOrbitFromOE(smd.Earth.SOI, 0, 0.0, 0, 0, 180, smd.Earth)
		_, _, tof := smd.Hohmann(initOrbit.RNorm(), initOrbit.VNorm(), tgtOrbit.RNorm(), tgtOrbit.VNorm(), smd.Earth)
		fmt.Printf("[%d] %s (~ %f days)\n", i, tof, tof.Hours()/24)
	}
}
