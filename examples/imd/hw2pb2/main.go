package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func main() {
	// Dawn orbit: rA = 2.57*smd.AU, rP = 2.17*smd.AU, and it departed at rP.
	// Dawn encounters Ceres when the latter is at its perihelion.
	// Compute information about Ceres
	aCeres := math.Pow(math.Pow(1682*24*3600, 2)*smd.Sun.GM()/(4*math.Pow(math.Pi, 2)), 1/3.)
	rPCeres := aCeres * (1 - 0.0758)
	fmt.Printf("Ceres: a=%.3f km\trP=%.3f km\n", aCeres, rPCeres)
	// Compute Dawn transfer trajectory information
	rATr := 2.57 * smd.AU
	rPTr := 2.17 * smd.AU
	aTr, eTr := smd.Radii2ae(rATr, rPTr)
	// Semi parameter
	pTr := aTr * (1 - math.Pow(eTr, 2))
	// True anomaly at encounters
	νEnc := math.Acos((1 / eTr) * (pTr/rPCeres - 1))
	// Create the Orbit object at that position
	trOrbitAtEnc := smd.NewOrbitFromOE(aTr, eTr, 0, 0, 0, νEnc*180/math.Pi, smd.Sun)
	fmt.Printf("transfer orbit at encounter: %s\n", trOrbitAtEnc)
	// Compute the mean motion:
	sinE, cosE := trOrbitAtEnc.SinCosE()
	E := math.Atan2(sinE, cosE)
	// Compute mean anomaly
	M := E - eTr*sinE
	// Compute mean motion
	nTr := (2 * math.Pi) / trOrbitAtEnc.Period().Seconds()
	// Compute the time since periapsis, since that's when Dawn departed Vesta
	tPeri, _ := time.ParseDuration(fmt.Sprintf("%fs", M/nTr))
	fmt.Printf("t = %s (~%.2f days or ~%.3f years)\n", tPeri, tPeri.Hours()/24, tPeri.Hours()/(24*365.25))
}
