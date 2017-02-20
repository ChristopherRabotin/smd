package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
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

	// Define the stations
	st1 := NewStation("st1", 0, -35.398333, 148.981944)
	st2 := NewStation("st2", 0, 40.427222, 355.749444)
	st3 := NewStation("st3", 0, 35.247164, 243.205)
	stations := []Station{st1, st2, st3}

	// Noise generation
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	σρ := 1e-3 // m , but all measurements in km.
	ρNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρ}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	σρDot := 1e-6 // mm/s , but all measurements in km/s.
	ρDotNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρDot}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}

	// Vector of measurements
	measurements := []smd.MissionState{}

	// Define the special export functions
	export := smd.ExportConfig{Filename: "LEO", Cosmo: false, AsCSV: true, Timestamp: false}
	export.CSVAppendHdr = func() string {
		hdr := "secondsSinceEpoch,"
		for _, st := range stations {
			hdr += fmt.Sprintf("%sRange,%sRangeRate,%sNoisyRange,%sNoisyRangeRate,", st.name, st.name, st.name, st.name)
		}
		fmt.Println(hdr)
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
				// Add this to the list of measurements
				measurements = append(measurements, state)
				vDiffECEF := make([]float64, 3)
				for i := 0; i < 3; i++ {
					vDiffECEF[i] = (vECEF[i] - st.V[i]) / ρ
				}
				// SC is visible.
				ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
				str += fmt.Sprintf("%f,%f,%f,%f,", ρ, ρDot, ρ+ρNoise.Rand(nil)[0], ρDot+ρDotNoise.Rand(nil)[0])
			} else {
				str += ",,,,"
			}
		}
		return str[:len(str)-1] // Remove trailing comma
	}

	// Generate the perturbed orbit
	smd.NewMission(smd.NewEmptySC("LEO", 0), leo, startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 3}, export).Propagate()

	// Take care of the measurements:
	fmt.Printf("Now have %d measurements\n", len(measurements))
}
