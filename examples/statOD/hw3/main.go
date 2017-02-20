package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

var wg sync.WaitGroup

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
	measurements := []Measurement{}

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
		// The station vectors are in ECEF, so let's convert the state to ECEF.
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
				str += fmt.Sprintf("%f,%f,%f,%f,", ρ, ρDot, ρ+ρNoise.Rand(nil)[0], ρDot+ρDotNoise.Rand(nil)[0])
				// Add this to the list of measurements
				measurements = append(measurements, Measurement{ρ, ρDot, θgst, state, st})
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

	// Perturbations in the estimate
	perts := smd.Perturbations{Jn: 3}

	// Initialize the KF
	// Noise
	Q := mat64.NewSymDense(6, nil)
	R := mat64.NewSymDense(2, nil)
	noiseKF := gokalman.NewNoiseless(Q, R)
	// Vanilla KF
	//var vanillaKF *gokalman.Vanilla

	// Take care of measurements.
	var prevState smd.MissionState
	var prevVEst gokalman.Estimate
	var vanillaKF *gokalman.Vanilla
	vanillaEstChan := make(chan (gokalman.Estimate), 1)
	go processEst("vanilla", vanillaEstChan)

	for i, measurement := range measurements {
		if i == 0 {
			prevState = measurement.State
			continue
		}
		orbitEstimate := smd.NewOrbitEstimate(fmt.Sprintf("est-%d", i), prevState.Orbit, perts, prevState.DT, measurement.State.DT.Sub(prevState.DT), time.Second)
		var P0 mat64.Symmetric
		if prevVEst != nil {
			// Initialize with the previous covariance
			P0 = prevVEst.Covariance()
		} else {
			P0 = gokalman.Identity(6)
		}
		// Initialize the KF with the first measurement as the state.
		// Let's re-create the state from the orbitEstimate, which also has Φ.
		x0 := mat64.NewVector(6, orbitEstimate.GetState()[0:6])
		if vanillaKF == nil {
			// Only start the KF once.
			var err error
			vanillaKF, _, err = gokalman.NewVanilla(x0, P0, orbitEstimate.Φ, mat64.NewDense(1, 1, nil), measurement.HTilde(), noiseKF)
			if err != nil {
				panic(err)
			}
		} else {
			// Update the matrices
			vanillaKF.SetMeasurementMatrix(measurement.HTilde())
			vanillaKF.SetStateTransition(orbitEstimate.Φ)
		}
		// Measurement update
		vest, err := vanillaKF.Update(measurement.StateVector(), mat64.NewVector(1, nil))
		if err != nil {
			fmt.Printf("[ERROR] %s %s\n", measurement, err)
			continue
		}
		prevVEst = vest
		vanillaEstChan <- vest
	}
	close(vanillaEstChan)
	wg.Wait()

}

func processEst(fn string, estChan chan (gokalman.Estimate)) {
	wg.Add(1)
	ce, _ := gokalman.NewCSVExporter([]string{"x", "y", "z", "xDot", "yDot", "zDot"}, ".", fn+".csv")
	for {
		est, more := <-estChan
		if !more {
			//oe.Close()
			ce.Close()
			wg.Done()
			break
		}
		ce.Write(est)
	}
}
