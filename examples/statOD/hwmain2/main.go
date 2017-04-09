package main

import (
	"flag"
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
	ekfTrigger     = 15    // Number of measurements prior to switching to EKF mode.
	ekfDisableTime = -1200 // Seconds between measurements to switch back to CKF. Set as negative to ignore.
	sncEnabled     = false // Set to false to disable SNC.
	sncDisableTime = 1200  // Number of seconds between measurements to skip using SNC noise.
	sncRIC         = false // Set to true if the noise should be considered defined in PQW frame.
	timeBasedPlot  = false // Set to true to plot time, or false to plot on measurements.
	smoothing      = false // Set to true to smooth the CKF.
)

var (
	σQExponent float64
	wg         sync.WaitGroup
)

var debug = flag.Bool("debug", false, "verbose debug")

func init() {
	flag.Float64Var(&σQExponent, "sigmaExp", 6, "exponent for the Q sigma (default is 6, so sigma=1e-6).")
}

func main() {
	flag.Parse()
	// Define the times
	startDT := time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)
	endDT := startDT.Add(time.Duration(24) * time.Hour)
	// Define the orbits
	leo := smd.NewOrbitFromOE(7000, 0.001, 30, 80, 40, 0, smd.Earth)

	// Define the stations
	σρ := math.Pow(1e-3, 2)    // m , but all measurements in km.
	σρDot := math.Pow(1e-3, 2) // m/s , but all measurements in km/s.
	st1 := smd.NewStation("st1", 0, 10, -35.398333, 148.981944, σρ, σρDot)
	st2 := smd.NewStation("st2", 0, 10, 40.427222, 355.749444, σρ, σρDot)
	st3 := smd.NewStation("st3", 0, 10, 35.247164, 243.205, σρ, σρDot)
	stations := []smd.Station{st1, st2, st3}

	measurements := make(map[time.Time]smd.Measurement)
	measurementTimes := []time.Time{}
	numMeasurements := 0 // Easier to count them here than to iterate the map to count.

	// Define the special export functions
	export := smd.ExportConfig{Filename: "LEO", Cosmo: false, AsCSV: true, Timestamp: false}
	export.CSVAppendHdr = func() string {
		hdr := "secondsSinceEpoch,"
		for _, st := range stations {
			hdr += fmt.Sprintf("%sRange,%sRangeRate,%sNoisyRange,%sNoisyRangeRate,", st.Name, st.Name, st.Name, st.Name)
		}
		return hdr[:len(hdr)-1] // Remove trailing comma
	}
	export.CSVAppend = func(state smd.State) string {
		Δt := state.DT.Sub(startDT).Seconds()
		str := fmt.Sprintf("%f,", Δt)
		θgst := Δt * smd.EarthRotationRate
		roundedDT := state.DT.Truncate(time.Second)
		// Compute visibility for each station.
		for _, st := range stations {
			measurement := st.PerformMeasurement(θgst, state)
			if measurement.Visible {
				// Sanity check
				if _, exists := measurements[roundedDT]; exists {
					panic(fmt.Errorf("already have a measurement for %s", state.DT))
				}
				measurements[roundedDT] = measurement
				measurementTimes = append(measurementTimes, roundedDT)
				numMeasurements++
				str += measurement.CSV()
			} else {
				str += ",,,,"
			}
		}
		return str[:len(str)-1] // Remove trailing comma
	}

	// Generate the true orbit -- Mtrue
	timeStep := 10 * time.Second
	scName := "LEO"
	smd.NewPreciseMission(smd.NewEmptySC(scName, 0), leo, startDT, endDT, smd.Perturbations{Jn: 3}, timeStep, false, export).Propagate()

	// Take care of the measurements:
	fmt.Printf("\n[INFO] Generated %d measurements\n", numMeasurements)
	// Let's mark those as the truth so we can plot that.
	stateTruth := make([]*mat64.Vector, len(measurements))
	truthMeas := make([]*mat64.Vector, len(measurements))
	for measNo, measTime := range measurementTimes {
		measurement := measurements[measTime]
		stateTruth[measNo] = measurement.State.Vector()
		truthMeas[measNo] = measurement.StateVector()
	}
	truth := gokalman.NewBatchGroundTruth(stateTruth, truthMeas)

	// Compute number of states which will be generated.
	numStates := int((measurementTimes[len(measurementTimes)-1].Sub(measurementTimes[0])).Seconds()/timeStep.Seconds()) + 1
	residuals := make([]*mat64.Vector, numStates)

	// Get the first measurement as an initial orbit estimation.
	firstDT := measurementTimes[0]
	estOrbit := measurements[firstDT].State.Orbit
	startDT = firstDT //.Add(-10 * time.Second)
	// TODO: Add noise to initial orbit estimate.

	// Perturbations in the estimate
	estPerts := smd.Perturbations{Jn: 2}

	stateEstChan := make(chan (smd.State))
	mEst := smd.NewPreciseMission(smd.NewEmptySC(scName+"Est", 0), &estOrbit, startDT, startDT.Add(-1), estPerts, timeStep, true, smd.ExportConfig{})
	mEst.RegisterStateChan(stateEstChan)

	// Go-routine to advance propagation.
	go mEst.PropagateUntil(measurementTimes[len(measurementTimes)-1], true)

	// KF filter initialization stuff.

	// Initialize the KF noise
	σQx := math.Pow(10, -2*σQExponent)
	var σQy, σQz float64
	if !sncRIC {
		σQy = σQx
		σQz = σQx
	}
	noiseQ := mat64.NewSymDense(3, []float64{σQx, 0, 0, 0, σQy, 0, 0, 0, σQz})
	noiseR := mat64.NewSymDense(2, []float64{σρ, 0, 0, σρDot})
	noiseKF := gokalman.NewNoiseless(noiseQ, noiseR)

	// Take care of measurements.
	estHistory := make([]*gokalman.HybridKFEstimate, len(measurements))
	stateHistory := make([]*mat64.Vector, len(measurements)) // Stores the histories of the orbit estimate (to post compute the truth)
	estChan := make(chan (gokalman.Estimate), 1)
	go processEst("hybridkf", estChan)

	prevP := mat64.NewSymDense(6, nil)
	var covarDistance float64 = 50
	var covarVelocity float64 = 1
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}

	visibilityErrors := 0

	if smoothing {
		fmt.Println("[INFO] Smoothing enabled")
	}

	if ekfTrigger < 0 {
		fmt.Println("[WARNING] EKF disabled")
	} else {
		if smoothing {
			fmt.Println("[ERROR] Enabling smooth has NO effect because EKF is enabled")
		}
		if ekfTrigger < 10 {
			fmt.Println("[WARNING] EKF may be turned on too early")
		} else {
			fmt.Printf("[INFO] EKF will turn on after %d measurements\n", ekfTrigger)
		}
	}

	var prevStationName = ""
	var prevDT time.Time
	var ckfMeasNo = 0
	measNo := 1
	stateNo := 0
	kf, _, err := gokalman.NewHybridKF(mat64.NewVector(6, nil), prevP, noiseKF, 2)
	if err != nil {
		panic(fmt.Errorf("%s", err))
	}
	// Now let's do the filtering.
	for {
		state, more := <-stateEstChan
		if !more {
			break
		}
		stateNo++
		roundedDT := state.DT.Truncate(time.Second)
		measurement, exists := measurements[roundedDT]
		if !exists {
			if measNo == 0 {
				time.Sleep(time.Second)
				panic(fmt.Errorf("should start KF at first measurement: \n%s (got)\n%s (exp)", roundedDT, measurementTimes[0]))
			}
			// There is no truth measurement here, let's only predict the KF covariance.
			kf.Prepare(state.Φ, nil)
			est, perr := kf.Predict()
			if perr != nil {
				panic(fmt.Errorf("[ERR!] (#%04d)\n%s", measNo, perr))
			}
			// TODO: Plot this too.
			stateEst := mat64.NewVector(6, nil)
			stateEst.SubVec(est.State(), state.Vector())
			// NOTE: The state seems to be all I need, along with Phi maybe (?) because the KF already uses the previous state?!
			if *debug {
				fmt.Printf("[pred] (%04d) %+v\n", measNo, mat64.Formatted(est.State().T()))
			}
			estChan <- truth.ErrorWithOffset(-1, est, nil)
			continue
		}
		if roundedDT != measurementTimes[measNo] {
			panic(fmt.Errorf("[ERR!] %04d delta = %s\tstate=%s\tmeas=%s", measNo, state.DT.Sub(measurementTimes[measNo]), state.DT, measurementTimes[measNo]))
		}

		if measNo == 0 {
			prevDT = measurement.State.DT
		}

		// Let's perform a full update since there is a measurement.
		ΔtDuration := measurement.State.DT.Sub(prevDT)
		Δt := ΔtDuration.Seconds() // Everything is in seconds.
		// Infomrational messages.
		if !kf.EKFEnabled() && ckfMeasNo == ekfTrigger {
			// Switch KF to EKF mode
			kf.EnableEKF()
			fmt.Printf("[info] #%04d EKF now enabled\n", measNo)
		} else if kf.EKFEnabled() && ekfDisableTime > 0 && Δt > ekfDisableTime {
			// Switch KF back to CKF mode
			kf.DisableEKF()
			ckfMeasNo = 0
			fmt.Printf("[info] #%04d EKF now disabled (Δt=%s)\n", measNo, ΔtDuration)
		}

		if measurement.Station.Name != prevStationName {
			fmt.Printf("[info] #%04d %s in visibility of %s (T+%s)\n", measNo, scName, measurement.Station.Name, measurement.State.DT.Sub(startDT))
			prevStationName = measurement.Station.Name
		}

		// Compute "real" measurement
		computedObservation := measurement.Station.PerformMeasurement(measurement.Timeθgst, state)
		if !computedObservation.Visible {
			fmt.Printf("[WARN] station %s should see the SC but does not\n", measurement.Station.Name)
			visibilityErrors++
		}

		Htilde := computedObservation.HTilde()
		kf.Prepare(state.Φ, Htilde)
		if sncEnabled {
			if Δt < sncDisableTime {
				if sncRIC {
					// Build the RIC DCM
					rUnit := smd.Unit(state.Orbit.R())
					cUnit := smd.Unit(state.Orbit.H())
					iUnit := smd.Unit(smd.Cross(rUnit, cUnit))
					dcmVals := make([]float64, 9)
					for i := 0; i < 3; i++ {
						dcmVals[i] = rUnit[i]
						dcmVals[i+3] = cUnit[i]
						dcmVals[i+6] = iUnit[i]
					}
					// Update the Q matrix in the PQW
					dcm := mat64.NewDense(3, 3, dcmVals)
					var QECI, QECI0 mat64.Dense
					QECI0.Mul(noiseQ, dcm.T())
					QECI.Mul(dcm, &QECI0)
					QECISym, err := gokalman.AsSymDense(&QECI)
					if err != nil {
						fmt.Printf("[ERR!] QECI is not symmertric!")
						panic(err)
					}
					kf.SetNoise(gokalman.NewNoiseless(QECISym, noiseR))
				}
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
			panic(fmt.Errorf("[ERR!] %s", err))
		}

		prevP = est.Covariance().(*mat64.SymDense)
		stateEst := mat64.NewVector(6, nil)
		stateEst.AddVec(state.Vector(), est.State())
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(Htilde, est.State())
		residual.AddScaledVec(residual, -1, est.ObservationDev())
		residual.ScaleVec(-1, residual)
		residuals[stateNo] = residual

		if smoothing {
			// Save to history in order to perform smoothing.
			estHistory[measNo] = est
			stateHistory[measNo] = stateEst
		} else {
			// Stream to CSV file
			estChan <- truth.ErrorWithOffset(measNo, est, state.Vector())
			diff := mat64.NewVector(6, nil)
			diff.SubVec(stateEst, measurement.State.Vector())
			if mat64.Norm(diff, 2) > 1 && *debug {
				fmt.Printf("[diff] (%04d) %+v\n", measNo, mat64.Formatted(diff.T()))
			}
		}
		prevDT = measurement.State.DT

		// If in EKF, update the reference trajectory.
		if kf.EKFEnabled() {
			// Update the state from the error.
			R, V := state.Orbit.RV()
			for i := 0; i < 3; i++ {
				R[i] += est.State().At(i, 0)
				V[i] += est.State().At(i+3, 0)
			}
			mEst.Orbit = smd.NewOrbitFromRV(R, V, smd.Earth)
		}
		ckfMeasNo++
		measNo++
	} // end while true

	close(estChan)
	wg.Wait()

	severity := "INFO"
	if visibilityErrors > 0 {
		severity = "WARNING"
	}
	fmt.Printf("[%s] %d visibility errors\n", severity, visibilityErrors)
	// Write the residuals to a CSV file
	f, ferr := os.Create("./hkf-residuals.csv")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()
	f.WriteString("rho,rhoDot\n")
	for _, residual := range residuals {
		csv := "0,0\n"
		if residual != nil {
			csv = fmt.Sprintf("%f,%f\n", residual.At(0, 0), residual.At(1, 0))
		}
		if _, err := f.WriteString(csv); err != nil {
			panic(err)
		}
	}

}

func processEst(fn string, estChan chan (gokalman.Estimate)) {
	wg.Add(1)
	// We also compute the RMS here.
	numMeasurements := 0
	rmsPosition := 0.0
	rmsVelocity := 0.0
	ce, _ := gokalman.NewCustomCSVExporter([]string{"x", "y", "z", "xDot", "yDot", "zDot"}, ".", fn+".csv", 3)
	for {
		est, more := <-estChan
		if !more {
			ce.Close()
			wg.Done()
			break
		}
		numMeasurements++
		for i := 0; i < 3; i++ {
			rmsPosition += math.Pow(est.State().At(i, 0), 2)
			rmsVelocity += math.Pow(est.State().At(i+3, 0), 2)
		}
		ce.Write(est)
	}
	// Compute RMS.
	rmsPosition /= float64(numMeasurements)
	rmsVelocity /= float64(numMeasurements)
	rmsPosition = math.Sqrt(rmsPosition)
	rmsVelocity = math.Sqrt(rmsVelocity)
	fmt.Printf("=== RMS ===\nPosition = %f\tVelocity = %f\n", rmsPosition, rmsVelocity)
}
