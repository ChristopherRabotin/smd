package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

func main() {
	// Define the times
	startDT := time.Now()
	endDT := startDT.Add(time.Duration(24) * time.Hour)
	// Define the orbits
	leo := smd.NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, smd.Earth)
	geo := smd.NewOrbitFromOE(42163, 1e-5, 1, 70, 0, 180, smd.Earth)
	R := []float64{-7737.559071593195, -43881.87809094457, 0.0}
	V := []float64{3.347424567061589, 3.828541915617483, 0.0}
	hyp := smd.NewOrbitFromRV(R, V, smd.Earth)

	// Define the stations
	st1 := NewStation("st1", 0, -35.398333, 148.981944)
	st2 := NewStation("st2", 0, 40.427222, 355.749444)
	st3 := NewStation("st3", 0, 35.247164, 243.205)
	stations := []Station{st1, st2, st3}

	// Run test cases.
	for _, tcase := range []struct {
		name  string
		orbit *smd.Orbit
	}{{"Leo", leo}, {"Geo", geo}, {"Hyperbolic", hyp}} {
		fmt.Printf("==== %s =====\n", tcase.name)
		// Define the special export functions
		export := smd.ExportConfig{Filename: tcase.name, Cosmo: false, AsCSV: true, Timestamp: false}
		export.CSVAppendHdr = func() string {
			hdr := "secondsSinceEpoch,"
			for _, st := range stations {
				hdr += fmt.Sprintf("%sRange,%sRangeRate,", st.name, st.name)
			}
			return hdr[:len(hdr)-1] // Remove trailing comma
		}
		export.CSVAppend = func(state smd.MissionState) string {
			str := fmt.Sprintf("%f,", state.DT.Sub(startDT).Seconds())
			θgst := state.DT.Sub(startDT).Seconds() * smd.EarthRotationRate
			rECEF := smd.ECI2ECEF(state.Orbit.R(), θgst)
			vECEF := smd.ECI2ECEF(state.Orbit.V(), θgst)
			// Compute visibility for each station.
			for _, st := range stations {
				ρECEF, ρ, el, _ := st.RangeElAz(rECEF)
				if el >= 10 {
					vDiffECEF := make([]float64, 3)
					for i := 0; i < 3; i++ {
						vDiffECEF[i] = (vECEF[i] - st.V[i]) / ρ
					}
					// SC is visible.
					ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
					str += fmt.Sprintf("%f,%f,", ρ, ρDot)
				} else {
					str += ",,"
				}
			}
			return str[:len(str)-1] // Remove trailing comma
		}

		// Generate the orbits
		smd.NewMission(smd.NewEmptySC(tcase.name, 0), tcase.orbit, startDT, endDT, smd.Cartesian, smd.Perturbations{}, export).Propagate()
	}
}

/*
R,V sc to ECEF
Convert station to ECEF
Compute velocity of station in ECEF using [0, 0, rot_earth] X station position in ECEF
Compute range in SEZ frame, *AND* get elevation and azimuth angles
If elevation is below 10, discard.
*/
