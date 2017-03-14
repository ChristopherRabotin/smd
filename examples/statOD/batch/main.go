package main

import (
	"fmt"
	"math"
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
	leo := smd.NewOrbitFromOE(7000, 0.001, 30, 80, 40, 0, smd.Earth)
	initR, initV := leo.RV()
	stateVector := mat64.NewVector(6, nil)
	for i := 0; i < 3; i++ {
		stateVector.SetVec(i, initR[i])
		stateVector.SetVec(i+3, initV[i])
	}

	// Define the stations
	σρ := math.Pow(1e-3, 2)    // m , but all measurements in km.
	σρDot := math.Pow(1e-3, 2) // m/s , but all measurements in km/s.
	st1 := NewStation("st1", 0, -35.398333, 148.981944, σρ, σρDot)
	st2 := NewStation("st2", 0, 40.427222, 355.749444, σρ, σρDot)
	st3 := NewStation("st3", 0, 35.247164, 243.205, σρ, σρDot)
	stations := []Station{st1, st2, st3}

	// Vector of measurements
	measurements := []Measurement{}
	// Vector of difference between truth and corrected orbits
	Δstate := []*mat64.Vector{}
	firstMeasEncountered := false
	firstMeasDT := time.Now() // Temporary value

	// Define the special export functions
	export := smd.ExportConfig{Filename: "LEO", Cosmo: true, AsCSV: true, Timestamp: false}
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
			_, measurement := st.PerformMeasurement(θgst, state)
			if measurement.Visible {
				if !firstMeasEncountered {
					firstMeasEncountered = true
					firstMeasDT = state.DT
				}
				measurements = append(measurements, measurement)
				str += measurement.CSV()
			} else {
				str += ",,,,"
			}
		}
		if firstMeasEncountered {
			// Add this orbit state to the list of states.
			R, V := state.Orbit.RV()
			state := mat64.NewVector(6, nil)
			for i := 0; i < 3; i++ {
				state.SetVec(i, R[i])
				state.SetVec(i+3, V[i])
			}
			Δstate = append(Δstate, state)
		}
		return str[:len(str)-1] // Remove trailing comma
	}

	timeStep := 2 * time.Second

	// Generate the perturbed orbit
	scName := "LEO"
	smd.NewPreciseMission(smd.NewEmptySC(scName, 0), leo, startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 3}, timeStep, export).Propagate()

	// Take care of the measurements:
	fmt.Printf("\n[INFO] Generated %d measurements\n", len(measurements))

	// Perturbations in the estimate
	estPerts := smd.Perturbations{Jn: 2}

	// Initialize the KF noise
	noiseQ := mat64.NewSymDense(3, nil)
	noiseR := mat64.NewSymDense(2, []float64{σρ, 0, 0, σρDot})
	noiseKF := gokalman.NewNoiseless(noiseQ, noiseR)

	visibilityErrors := 0
	var orbitEstimate *smd.OrbitEstimate

	kf := gokalman.NewBatchKF(len(measurements), noiseKF)
	var prevStationName = ""
	var prevΦ *mat64.Dense
	for measNo, measurement := range measurements {
		if !measurement.Visible {
			panic("why is there a non visible measurement?!")
		}
		if measNo == 0 {
			orbitEstimate = smd.NewOrbitEstimate("estimator", measurement.State.Orbit, estPerts, measurement.State.DT, time.Second)
		}
		prevΦ = orbitEstimate.Φ
		// Propagate the reference trajectory until the next measurement time.
		orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(ti+1, ti)

		// Compute Φ(ti+1, t0)
		var prevΦinv mat64.Dense
		if err := prevΦinv.Inverse(prevΦ); err != nil {
			panic(fmt.Errorf("the following Φ is singular:\n%+v", mat64.Formatted(prevΦ)))
		}
		var Φtit0 mat64.Dense
		Φtit0.Mul(orbitEstimate.Φ, &prevΦinv)

		if measurement.Station.name != prevStationName {
			fmt.Printf("[INFO] #%04d %s in visibility of %s (T+%s)\n", measNo, scName, measurement.Station.name, measurement.State.DT.Sub(startDT))
			prevStationName = measurement.Station.name
		}

		// Compute "real" measurement
		vis, computed := measurement.Station.PerformMeasurement(measurement.θgst, orbitEstimate.State())
		if !vis {
			fmt.Printf("[WARNING] station %s should see the SC but does not\n", measurement.Station.name)
			visibilityErrors++
		}
		Htilde := measurement.HTilde(orbitEstimate.State(), measurement.θgst)
		// Compute H
		var H mat64.Dense
		H.Mul(Htilde, &Φtit0)
		kf.SetNextMeasurement(measurement.Observation(), computed.Observation(), orbitEstimate.Φ, &H)
	}
	severity := "INFO"
	if visibilityErrors > 0 {
		severity = "WARNING"
	}
	fmt.Printf("[%s] %d visibility errors\n", severity, visibilityErrors)
	// Solve Batch
	_, P0, err := kf.Solve()
	if err != nil {
		panic(fmt.Errorf("could not solve BatchKF: %s", err))
	}
	fmt.Printf("Batch P0:\n%+v\n", mat64.Formatted(P0))
	// Let's perform the correction on the reference trajectory, and propagate it.
	//stateVector.SubVec(stateVector, xHat0)
	// Generate the new orbit via Mission.
	correctedOrbit := smd.NewOrbitFromRV([]float64{stateVector.At(0, 0), stateVector.At(1, 0), stateVector.At(2, 0)}, []float64{stateVector.At(3, 0), stateVector.At(4, 0), stateVector.At(5, 0)}, smd.Earth)
	// And propagate it to generate the state vectors, starting from the first measurement time.
	//startMeasDT := measurements[0].State.DT
	//endMeasDT := measurements[len(measurements)-1].State.DT

	// Generate the perturbed orbit. We'll use the export functions just to store the corrected state so we can compute the difference.
	scName = "CorrectedLEO"
	correctedExport := smd.ExportConfig{Filename: scName, Cosmo: true, AsCSV: true, Timestamp: false}
	correctedExport.CSVAppendHdr = func() string {
		return ""
	}
	exportIt := 0
	correctedExport.CSVAppend = func(state smd.MissionState) string {
		if !state.DT.Before(firstMeasDT) {
			// time.After is strict, and we need equality.
			R, V := state.Orbit.RV()
			state := mat64.NewVector(6, nil)
			for i := 0; i < 3; i++ {
				state.SetVec(i, R[i])
				state.SetVec(i+3, V[i])
			}
			initState := Δstate[exportIt]
			initState.SubVec(initState, state)
			Δstate[exportIt] = initState
			exportIt++
		}
		return ""
	}
	smd.NewPreciseMission(smd.NewEmptySC(scName, 0), correctedOrbit, firstMeasDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 3}, timeStep, correctedExport).Propagate()
	fmt.Printf("computed it=%d\tlen=%d\n", exportIt, len(Δstate))
	// Write the state errors to a file.
	f, err := os.Create("./batch-state-errors.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("\\Delta X,\\Delta Y,\\Delta Z,\\Delta X_{dot},\\Delta Y_{dot},\\Delta Z_{dot}\n")
	for _, delta := range Δstate {
		csv := fmt.Sprintf("%f,%f,%f,%f,%f,%f\n", delta.At(0, 0), delta.At(1, 0), delta.At(2, 0), delta.At(3, 0), delta.At(4, 0), delta.At(5, 0))
		if _, err := f.WriteString(csv); err != nil {
			panic(err)
		}
	}
}
