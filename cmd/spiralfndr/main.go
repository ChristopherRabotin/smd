package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ChristopherRabotin/smd"
)

const (
	debug = false
)

var (
	cpus     int
	planet   string
	stepSize float64
	wg       sync.WaitGroup
)

func init() {
	// Read flags
	flag.IntVar(&cpus, "cpus", -1, "number of CPUs to use for this simulation (set to 0 for max CPUs)")
	flag.StringVar(&planet, "planet", "undef", "departure planet to perform the spiral from")
	flag.Float64Var(&stepSize, "step", 15, "step size (10 to 30 recommended)")
}

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

func initEarthOrbit(i, Ω, ω, ν float64) *smd.Orbit {
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Earth)
}

// initMarsOrbit returns the initial orbit.
func initMarsOrbit(i, Ω, ω, ν float64) *smd.Orbit {
	// Exomars TGO.
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	return smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, smd.Mars)
}

func main() {
	flag.Parse()
	availableCPUs := runtime.NumCPU()
	if cpus <= 0 || cpus > availableCPUs {
		cpus = availableCPUs
	}
	runtime.GOMAXPROCS(cpus)
	fmt.Printf("running on %d CPUs\n", cpus)

	if stepSize <= 0 {
		fmt.Println("step size must be positive")
		flag.Usage()
		return
	} else if stepSize <= 5 {
		fmt.Println("[WARNING] A small step size will take several days to iterate over all possibilities")
	}

	var orbitPtr func(i, Ω, ω, ν float64) *smd.Orbit
	planet = strings.ToLower(planet)
	switch planet {
	case "mars":
		orbitPtr = initMarsOrbit
	case "earth":
		orbitPtr = initEarthOrbit
	default:
		fmt.Printf("unsupported planet `%s`\n", planet)
		flag.Usage()
		return
	}

	fmt.Printf("Finding spirals leaving from %s\n", planet)

	//name := "spiral-mars"
	depart := time.Date(2018, 11, 8, 0, 0, 0, 0, time.UTC)
	chgframePath := "../refframes/chgframe.py"
	maxV := -1e3
	minV := +1e3
	var maxOrbit smd.Orbit
	var minOrbit smd.Orbit
	a, e, _, _, _, _, _, _, _ := orbitPtr(10, 10, 10, 10).Elements()
	rslts := make(chan string, 10)
	wg.Add(1)
	go streamResults(a, e, fmt.Sprintf("%s-%.0fstep", planet, stepSize), rslts)
	for i := 1.0; i < 90; i += stepSize {
		for Ω := 0.0; Ω < 360; Ω += stepSize {
			for ω := 0.0; ω < 360; ω += stepSize {
				for ν := 0.0; ν < 360; ν += stepSize {
					initOrbit := orbitPtr(i, Ω, ω, ν)
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
					rslts <- fmt.Sprintf("%f,%f,%f,%f,%f\n", vNorm, i, Ω, ω, ν)
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
		}
	}
	fmt.Printf("\n\n=== RESULT ===\n\nmaxV=%.3f km/s\t%s\nminV=%.3f km/s\t%s\n\n", maxV, maxOrbit, minV, minOrbit)
}

func streamResults(a, e float64, fn string, rslts <-chan string) {
	// Write CSV file.
	f, err := os.Create(fmt.Sprintf("./%s.csv", fn))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	// Header
	f.WriteString(fmt.Sprintf("#a=%f km\te=%f\n#V(km/s), i (degrees), raan (degrees), arg peri (degrees),nu (degrees)\n", a, e))
	for {
		rslt, more := <-rslts
		if more {
			if _, err := f.WriteString(rslt); err != nil {
				panic(err)
			}
		} else {
			break
		}
	}
	f.Close()
	wg.Done()
}
