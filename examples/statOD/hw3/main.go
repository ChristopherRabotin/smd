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

const (
	ekfTrigger    = 15    // Number of measurements prior to switching to EKF mode.
	sncEnabled    = true  // Set to false to disable SNC.
	timeBasedPlot = false // Set to true to plot time, or false to plot on measurements.
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
			_, measurement := st.PerformMeasurement(θgst, state)
			if measurement.Visible {
				measurements = append(measurements, measurement)
				str += measurement.CSV()
			} else {
				str += ",,,,"
			}
		}
		return str[:len(str)-1] // Remove trailing comma
	}

	// Generate the perturbed orbit
	scName := "LEO"
	smd.NewPreciseMission(smd.NewEmptySC(scName, 0), leo, startDT, endDT, smd.Cartesian, smd.Perturbations{Jn: 3}, 2*time.Second, export).Propagate()

	// Take care of the measurements:
	fmt.Printf("\n[INFO] Generated %d measurements\n", len(measurements))
	// Let's mark those as the truth so we can plot that.
	stateTruth := make([]*mat64.Vector, len(measurements))
	truthMeas := make([]*mat64.Vector, len(measurements))
	residuals := make([]*mat64.Vector, len(measurements))
	for measNo, measurement := range measurements {
		orbit := make([]float64, 6)
		R, V := measurement.State.Orbit.RV()
		for i := 0; i < 3; i++ {
			orbit[i] = R[i]
			orbit[i+3] = V[i]
		}
		stateTruth[measNo] = mat64.NewVector(6, orbit)
		truthMeas[measNo] = measurement.StateVector()
	}
	truth := gokalman.NewBatchGroundTruth(stateTruth, truthMeas)

	// Perturbations in the estimate
	estPerts := smd.Perturbations{Jn: 3}

	// Initialize the KF noise
	σQ := math.Pow(1e-6, 2)
	Q := mat64.NewSymDense(3, []float64{σQ, 0, 0, 0, σQ, 0, 0, 0, σQ})
	R := mat64.NewSymDense(2, []float64{σρ, 0, 0, σρDot})
	noiseKF := gokalman.NewNoiseless(Q, R)

	// Take care of measurements.
	estChan := make(chan (gokalman.Estimate), 1)
	go processEst("hybridkf", estChan)

	prevXHat := mat64.NewVector(6, nil)
	prevP := mat64.NewSymDense(6, nil)
	var covarDistance float64 = 50
	var covarVelocity float64 = 1
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}

	visibilityErrors := 0
	var orbitEstimate *smd.OrbitEstimate

	if ekfTrigger < 0 {
		fmt.Println("[WARNING] EKF disabled")
	} else if ekfTrigger < 10 {
		fmt.Println("[WARNING] EKF may be turned on too early")
	} else {
		fmt.Printf("[INFO] EKF will turn on after %d measurements\n", ekfTrigger)
	}

	var kf *gokalman.HybridKF
	var prevStationName = ""
	var prevDT time.Time
	for measNo, measurement := range measurements {
		if !measurement.Visible {
			panic("why is there a non visible measurement?!")
		}
		if measNo == 0 {
			prevDT = measurement.State.DT
			orbitEstimate = smd.NewOrbitEstimate("estimator", measurement.State.Orbit, estPerts, measurement.State.DT, time.Second)
			var err error
			kf, _, err = gokalman.NewHybridKF(prevXHat, prevP, noiseKF, 2)
			if err != nil {
				panic(fmt.Errorf("%s", err))
			}
		} else if measNo == ekfTrigger {
			// Switch KF to EKF mode
			kf.EnableEKF()
			fmt.Printf("[INFO] #%04d EKF now enabled\n", measNo)
		}
		if timeBasedPlot {
			// Propagate and predict for each time step until next measurement.
			for prevDT.Before(measurement.State.DT) {
				nextDT := prevDT.Add(10 * time.Second)
				orbitEstimate.PropagateUntil(nextDT) // This leads to Φ(ti+1, ti)
				// Only do a prediction.
				kf.Prepare(orbitEstimate.Φ, nil)
				est, perr := kf.Predict()
				if perr != nil {
					panic(fmt.Errorf("[error] (#%04d)\n%s", measNo, perr))
				}
				stateEst := mat64.NewVector(6, nil)
				R, V := orbitEstimate.State().Orbit.RV()
				for i := 0; i < 3; i++ {
					stateEst.SetVec(i, R[i])
					stateEst.SetVec(i+3, V[i])
				}
				fmt.Printf("%s\n\n", est)
				//fmt.Printf("%+v\n", mat64.Formatted(stateEst.T()))
				estChan <- truth.ErrorWithOffset(measNo, est, stateEst)
				prevDT = nextDT
			}
			continue
		}
		ΔtDuration := measurement.State.DT.Sub(prevDT)
		Δt := ΔtDuration.Seconds() // Everything is in seconds.
		if Δt > 60 {
			fmt.Printf("[INFO] #%04d occurred %s after #%04d\n", measNo, ΔtDuration, measNo-1)
		}
		// Propagate the reference trajectory until the next measurement time.
		orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(ti+1, ti)

		if measurement.Station.name != prevStationName {
			fmt.Printf("[INFO] #%04d %s in visibility of %s (T+%s)\n", measNo, scName, measurement.Station.name, measurement.State.DT.Sub(startDT))
			prevStationName = measurement.Station.name
		}

		// Compute "real" measurement
		vis, computedObservation := measurement.Station.PerformMeasurement(measurement.θgst, orbitEstimate.State())
		if !vis {
			fmt.Printf("[WARNING] station %s should see the SC but does not\n", measurement.Station.name)
			visibilityErrors++
		}
		Htilde := measurement.HTilde(orbitEstimate.State(), measurement.θgst)
		kf.Prepare(orbitEstimate.Φ, Htilde)
		if sncEnabled {
			if Δt < 60*38 {
				// Only enable SNC for small time differences between measurements.
				Γtop := gokalman.ScaledDenseIdentity(3, math.Pow(Δt, 2)/2)
				Γbot := gokalman.ScaledDenseIdentity(3, Δt)
				Γ := mat64.NewDense(6, 3, nil)
				Γ.Stack(Γtop, Γbot)
				kf.PreparePNT(Γ)
			}
		}
		est, err := kf.Update(measurement.StateVector(), computedObservation.StateVector())
		if err != nil {
			panic(fmt.Errorf("[error] %s", err))
		}
		prevXHat = est.State()
		prevP = est.Covariance().(*mat64.SymDense)
		stateEst := mat64.NewVector(6, nil)
		R, V := orbitEstimate.State().Orbit.RV()
		for i := 0; i < 3; i++ {
			stateEst.SetVec(i, R[i])
			stateEst.SetVec(i+3, V[i])
		}
		stateEst.AddVec(stateEst, est.State())
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(Htilde, est.State())
		residual.AddScaledVec(residual, -1, est.ObservationDev())
		residual.ScaleVec(-1, residual)
		residuals[measNo] = residual

		// Stream to CSV file
		estChan <- truth.ErrorWithOffset(measNo, est, stateEst)
		prevDT = measurement.State.DT

		// If in EKF, update the reference trajectory.
		if kf.EKFEnabled() {
			// Update the state from the error.
			state := est.State()
			R, V := orbitEstimate.Orbit.RV()
			for i := 0; i < 3; i++ {
				R[i] += state.At(i, 0)
				V[i] += state.At(i+3, 0)
			}
			orbitEstimate = smd.NewOrbitEstimate("estimator", *smd.NewOrbitFromRV(R, V, smd.Earth), estPerts, measurement.State.DT, time.Second)
		}

	}
	close(estChan)
	wg.Wait()

	severity := "INFO"
	if visibilityErrors > 0 {
		severity = "WARNING"
	}
	fmt.Printf("[%s] %d visibility errors\n", severity, visibilityErrors)
	// Write the residuals to a CSV file
	fname := "hkf"
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
