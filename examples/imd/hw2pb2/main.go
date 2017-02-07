package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
)

/*
. NASA launched the Dawn spacecraft in September 2007 on a mission to study two protoplanets of
the asteroid belt, Vesta and Ceres. Dawn arrived at Vesta in July 2011 and stayed for approximately 14
months. When Dawn departed Vesta on September 5, 2012, it was on a heliocentric orbit with radius
of periapsis 2.17 AU and radius of apoapsis 2.57 AU. The protoplanet Ceres has a period of 1682 days
and an eccentricity of 0.0758. Determine the transfer time of flight using the following assumptions:
+ Dawn departed Vesta at periapsis of the transfer orbit.
+ Dawn will encounter Ceres when Ceres is at its perihelion.
+ Dawn is approaching apoapsis on the transfer orbit when it encounters Ceres.
*/

func main() {
	// Dawn orbit: rA = 2.57*smd.AU, rP = 2.17*smd.AU, and it departed at rP.
	// Dawn encounters Ceres when the latter is at its perihelion.
	// Compute information about Ceres
	aCeres := math.Pow(math.Pow(1682*24*3600, 2)*smd.Sun.GM()/(4*math.Pow(math.Pi, 2)), 1/3.)
	rPCeres := aCeres * (1 - 0.0758)
	fmt.Printf("Ceres: a=%.3f km\trP=%.3f km\n", aCeres, rPCeres)
	rATr := 2.57 * smd.AU
	rPTr := 2.17 * smd.AU
	aTr, eTr := smd.Radii2ae(rATr, rPTr)
	pTr := aTr * (1 - math.Pow(eTr, 2))
	νEnc := math.Acos((1 / eTr) * (pTr/rPCeres - 1))
	trOrbitAtEnc := smd.NewOrbitFromOE(aTr, eTr, 0, 0, 0, νEnc, smd.Sun)
	fmt.Printf("transfer orbit at encounter: %s\n", trOrbitAtEnc)
	// Compute the mean motion:
	sinE, cosE := trOrbitAtEnc.SinCosE()
	E := math.Atan2(sinE, cosE)
	M := E - eTr*sinE
	nTr := (2 * math.Pi) / trOrbitAtEnc.Period().Seconds()
	// Compute the time since periapsis, since that's when Dawn departed Vesta:
	tPeri, _ := time.ParseDuration(fmt.Sprintf("%fs", M/nTr))
	fmt.Printf("t = %s (~ %.2f days)\n", tPeri, tPeri.Hours()/24)
}
