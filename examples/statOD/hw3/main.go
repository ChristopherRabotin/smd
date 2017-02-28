package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

var (
	wg sync.WaitGroup
)

func main() {
	// Define the times
	startDT := time.Now()
	endDT := startDT.Add(time.Duration(24) * time.Hour)
	// Define the orbits
	leo := smd.NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, smd.Earth)

	// Define the stations
	σρ := 1e-3    // m , but all measurements in km.
	σρDot := 1e-6 // mm/s , but all measurements in km/s.
	st1 := NewStation("st1", 0, -35.398333, 148.981944, σρ, σρDot)
	st2 := NewStation("st2", 0, 40.427222, 355.749444, σρ, σρDot)
	st3 := NewStation("st3", 0, 35.247164, 243.205, σρ, σρDot)
	stations := []Station{st1, st2, st3}

	// Vector of measurements
	measurements := []Measurement{}

	// Define the special export functions
	export := smd.ExportConfig{Filename: "LEO", Cosmo: false, AsCSV: true, Timestamp: false}
	export.CSVAppendHdr = func() string {
		hdr := "secondsSinceEpoch,"
		for _, st := range stations {
			hdr += fmt.Sprintf("%sRange,%sRangeRate,%sNoisyRange,%sNoisyRangeRate,", st.name, st.name, st.name, st.name)
		}
		return hdr[:len(hdr)-1] // Remove trailing comma
	}
	export.CSVAppend = func(state smd.MissionState) string {
		Δt := state.DT.Sub(startDT).Seconds()
		str := fmt.Sprintf("%f,", Δt)
		θgst := Δt * smd.EarthRotationRate
		// Compute visibility for each station.
		for _, st := range stations {
			visible, measurement := st.PerformMeasurement(θgst, state)
			if visible {
				measurements = append(measurements, measurement)
				str += measurement.CSV()
			} else {
				str += ",,,,"
			}
		}
		return str[:len(str)-1] // Remove trailing comma
	}

	// Generate the perturbed orbit
	smd.NewMission(smd.NewEmptySC("LEO", 0), leo, startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 2}, export).Propagate()

	// Take care of the measurements:
	fmt.Printf("Now have %d measurements\n", len(measurements))
	// Let's mark those as the truth so we can plot that.
	stateTruth := make([]*mat64.Vector, len(measurements))
	truthMeas := make([]*mat64.Vector, len(measurements))
	residuals := make([]*mat64.Vector, len(measurements))
	for i, measurement := range measurements {
		orbit := make([]float64, 6)
		R, V := measurement.State.Orbit.RV()
		for k := 0; k < 3; k++ {
			orbit[k] = R[k]
			orbit[k+3] = V[k]
		}
		stateTruth[i] = mat64.NewVector(6, orbit)
		truthMeas[i] = measurement.StateVector()
	}
	truth := gokalman.NewBatchGroundTruth(stateTruth, truthMeas)

	// Perturbations in the estimate
	estPerts := smd.Perturbations{Jn: 2}

	// Initialize the KF
	Q := mat64.NewSymDense(6, nil)
	R := mat64.NewSymDense(2, []float64{σρ, 0, 0, σρDot})
	noiseKF := gokalman.NewNoiseless(Q, R)

	// Take care of measurements.
	estChan := make(chan (gokalman.Estimate), 1)
	go processEst("hybridckf", estChan)

	prevXHat := mat64.NewVector(6, nil)
	prevP := mat64.NewSymDense(6, nil)
	var covarDistance float64 = 50
	var covarVelocity float64 = 1
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}
	var prevθ float64

	visibilityErrors := 0
	var orbitEstimate *smd.OrbitEstimate

	var ckf *gokalman.HybridCKF
	var prevStationName = ""
	for i, measurement := range measurements {
		if measurement.Station.name != prevStationName {
			fmt.Printf("Now visible by %s (#%d)\n", measurement.Station.name, i)
			prevStationName = measurement.Station.name
		}

		if i == 0 {
			orbitEstimate = smd.NewOrbitEstimate("estimator", measurement.State.Orbit, estPerts, measurement.State.DT, time.Second)
			var err error
			ckf, _, err = gokalman.NewHybridCKF(prevXHat, prevP, noiseKF, 2)
			if err != nil {
				panic(fmt.Errorf("%s", err))
			}
		}

		// Propagate the reference trajectory until the next measurement time.
		orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(t0, ti)

		// Compute "real" measurement
		vis, computedObservation := measurement.Station.PerformMeasurement(measurement.θgst, orbitEstimate.State())
		if !vis {
			fmt.Printf("[warning] station %s should see the SC but does not\n", measurement.Station.name)
			visibilityErrors++
		}
		θdot := measurement.θgst - prevθ
		Htilde := measurement.HTilde(orbitEstimate.State(), measurement.θgst, θdot)
		ckf.Prepare(orbitEstimate.Φ, Htilde)
		est, err := ckf.Update(measurement.StateVector(), computedObservation.StateVector())
		if err != nil {
			panic(fmt.Errorf("[error] %s", err))
		}
		prevXHat = est.State()
		prevP = est.Covariance().(*mat64.SymDense)
		stateEst := mat64.NewVector(6, nil)
		R, V := orbitEstimate.State().Orbit.RV()
		for x := 0; x < 3; x++ {
			stateEst.SetVec(x, R[x])
			stateEst.SetVec(x+3, V[x])
		}
		stateEst.AddVec(stateEst, est.State())
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(Htilde, est.State())
		residual.AddScaledVec(residual, -1, est.ObservationDev())
		residual.ScaleVec(-1, residual)
		residuals[i] = residual

		// Stream to CSV file
		estChan <- truth.ErrorWithOffset(i, est, stateEst)

	}
	close(estChan)
	wg.Wait()

	fmt.Printf("\n%d visibility errors\n", visibilityErrors)
	// Write the residuals to a CSV file
	fname := "hckf"
	f, err := os.Create(fmt.Sprintf("./%s-residuals.csv", fname))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("rho,rhoDot\n")
	for _, residual := range residuals {
		csv := fmt.Sprintf("%f,%f\n", residual.At(0, 0), residual.At(1, 0))
		if _, err := f.WriteString(csv); err != nil {
			panic(err)
		}
	}
}

func processEst(fn string, estChan chan (gokalman.Estimate)) {
	wg.Add(1)
	ce, _ := gokalman.NewCustomCSVExporter([]string{"x", "y", "z", "xDot", "yDot", "zDot"}, ".", fn+".csv", 3)
	for {
		est, more := <-estChan
		if !more {
			ce.Close()
			wg.Done()
			break
		}
		ce.Write(est)
	}
}
