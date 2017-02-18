package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
)

/*
 * This example shows how to find the greatest heliocentric velocity at the end of a spiral by iterating on the initial
 * true anomaly.
 */

func sc() *smd.Spacecraft {
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	return smd.NewSpacecraft("Spiral", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{smd.NewToHyperbolic(nil)})
}

func initEarthOrbit() *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	ω := 10.0
	Ω := 5.0
	ν := 0.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// initMarsOrbit returns the initial orbit.
func initMarsOrbit(ν float64) *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	i := 10.0
	ω := 1.0
	Ω := 1.0
	//ν := 15.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

func main() {
	//name := "spiral-mars"
	depart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	chgframePath := "../../cmd/refframe/chgframe.py"
	//maxV := 0.0
	for ν := 0.0; ν < 360; ν++ {
		initOrbit := initMarsOrbit(ν)
		astro := smd.NewMission(sc(), initOrbit, depart, depart.Add(-1), smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{ /*Filename: name, AsCSV: false, Cosmo: true, Timestamp: false*/ })
		astro.Propagate()
		// We're now done so let's convert the position and velocity to heliocentric and check the output.
		R, V := initOrbit.RV()
		state := fmt.Sprintf("[%f,%f,%f,%f,%f,%f]", R[0], R[1], R[2], V[0], V[1], V[2])
		cmd := exec.Command(chgframePath, "-t", "J2000", "-f", "IAU_Mars", "-e", astro.CurrentDT.String(), "-s", state)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		newState := out.String()
		// Cf. https://play.golang.org/p/g-a4idjhIb
		newState = newState[1 : len(newState)-1]
		components := strings.Split(newState, ",")
		var nR = make([]float64, 3)
		var nV = make([]float64, 3)
		for i := 0; i < 6; i++ {
			fl, err := strconv.ParseFloat(strings.TrimSpace(components[i]), 64)
			if err != nil {
				panic(err)
			}
			if i < 3 {
				nR[i] = fl
			} else {
				nV[i-3] = fl
			}
		}
		vNorm := math.Sqrt(math.Pow(nV[0], 2) + math.Pow(nV[1], 2) + math.Pow(nV[2], 2))
		fmt.Printf("ν=%f\t=>V=%+v\t|V|=%f", ν, nV, vNorm)
		break
	}
}

func init() {
	runtime.GOMAXPROCS(3)
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}
