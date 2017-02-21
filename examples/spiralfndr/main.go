package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
)

const (
	debug = false
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

func initEarthOrbit(ω, ν float64) *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	i := 28.0
	//ω := 10.0
	Ω := 5.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// initMarsOrbit returns the initial orbit.
func initMarsOrbit(i, Ω, ω, ν float64) *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	//i := 1.0
	//ω := 1.0
	//Ω := 1.0
	//ν := 15.0
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

func main() {
	//name := "spiral-mars"
	depart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	chgframePath := "../../cmd/refframes/chgframe.py"
	maxV := -1e3
	minV := +1e3
	var maxOrbit smd.Orbit
	var minOrbit smd.Orbit
	tsv := "%i (degrees), raan (degrees), arg peri (degrees),nu (degrees),V(km/s)\n"
	stepSize := 30.
	for i := 0.0; i < 360; i += stepSize {
		for Ω := 0.0; Ω < 360; Ω += stepSize {
			for ω := 0.0; ω < 360; ω += stepSize {
				for ν := 0.0; ν < 360; ν += stepSize {
					initOrbit := initMarsOrbit(i, Ω, ω, ν)
					astro := smd.NewMission(sc(), initOrbit, depart, depart.Add(-1), smd.Cartesian, smd.Perturbations{}, smd.ExportConfig{})
					astro.Propagate()

					// Run chgframe
					// We're now done so let's convert the position and velocity to heliocentric and check the output.
					R, V := initOrbit.RV()
					state := fmt.Sprintf("[%f,%f,%f,%f,%f,%f]", R[0], R[1], R[2], V[0], V[1], V[2])
					if debug {
						fmt.Printf("\n=== RUNNING CMD ===\npython %s -t J2000 -f IAU_Earth -e \"%s\" -s \"%s\"\n", chgframePath, astro.CurrentDT.Format(time.ANSIC), state)
					}
					cmd := exec.Command("python", chgframePath, "-t", "J2000", "-f", "IAU_Mars", "-e", astro.CurrentDT.Format(time.ANSIC), "-s", state)
					cmdOut, err := cmd.Output()
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error converting orbit to helio ", err)
						os.Exit(1)
					}
					out := string(cmdOut)

					// Process output
					newState := strings.TrimSpace(string(out))
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
					// Add to TSV file
					tsv += fmt.Sprintf("%f,%f,%f,%f,%f\n", i, Ω, ω, ν, vNorm)
					if vNorm > maxV {
						maxV = vNorm
						maxOrbit = *initMarsOrbit(i, Ω, ω, ν)
					} else if vNorm < minV {
						minV = vNorm
						minOrbit = *initMarsOrbit(i, Ω, ω, ν)
					}
					if debug {
						fmt.Printf("\nν=%f\t=>V=%+v\t|V|=%f\n", ν, nV, vNorm)
					}
				}
			}
			fmt.Printf("\n\n=== RESULT ===\n\nmaxV=%.3f km/s\t%s\nminV=%.3f km/s\t%s\n\n", maxV, maxOrbit, minV, minOrbit)
			// Write CSV file.
			f, err := os.Create("./results.csv")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if _, err := f.WriteString(tsv); err != nil {
				panic(err)
			}
		}
	}
}
func init() {
	runtime.GOMAXPROCS(3)
}
