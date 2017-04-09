package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/soniakeys/meeus/julian"
)

const (
	ekfTrigger     = -15   // Number of measurements prior to switching to EKF mode.
	ekfDisableTime = -1200 // Seconds between measurements to switch back to CKF. Set as negative to ignore.
	sncEnabled     = false // Set to false to disable SNC.
	sncDisableTime = 1200  // Number of seconds between measurements to skip using SNC noise.
	sncRIC         = false // Set to true if the noise should be considered defined in PQW frame.
	smoothing      = true  // Set to true to smooth the CKF.
)

// Scenario constants
const (
	initJDE = 2456296.25
)

var (
	σQExponent float64
	measFile   string
	wg         sync.WaitGroup
)

var debug = flag.Bool("debug", false, "verbose debug")

func init() {
	flag.Float64Var(&σQExponent, "sigmaExp", 6, "exponent for the Q sigma (default is 6, so sigma=1e-6).")
	flag.StringVar(&measFile, "meas", "z", "measurement file number")
}

var (
	σρ              = math.Pow(5e-3, 2) // m , but all measurements in km.
	σρDot           = math.Pow(5e-6, 2) // m/s , but all measurements in km/s.
	_DSS34Canberra  = smd.NewStation("DSS34Canberra", 0.691750, 0, -35.398333, 148.981944, σρ, σρDot)
	_DSS65Madrid    = smd.NewStation("DSS65Madrid", 0.834939, 0, 40.427222, 355.749444, σρ, σρDot)
	_DSS13Goldstone = smd.NewStation("DSS13Goldstone", 1.07114904, 0, 35.247164, 243.205, σρ, σρDot)
)

func main() {
	flag.Parse()
	if measFile != "a" && measFile != "b" {
		log.Fatalf("unknown file `%s` (should be a or b)", measFile)
	}
	startDT := julian.JDToTime(initJDE)
	// Load measurements
	measurements, startDT, endDT := loadMeasurementFile(measFile, startDT)
	log.Printf("[info] Loaded %d measurements from %s to %s", len(measurements), startDT, endDT)

	timeStep := 10 * time.Second

	// Compute number of states which will be generated.
	numStates := int(endDT.Sub(startDT).Seconds()/timeStep.Seconds()) + 1
	residuals := make([]*mat64.Vector, numStates)

	// Get the first measurement as an initial orbit estimation.
	firstDT := startDT
	estOrbit := smd.NewOrbitFromRV([]float64{-274096790.0, -92859240.0, -40199490.0}, []float64{32.67, -8.94, -3.88}, smd.Earth)
	startDT = firstDT
	// Perturbations in the estimate
	estPerts := smd.Perturbations{PerturbingBody: &smd.Sun}

	stateEstChan := make(chan (smd.State))
	mEst := smd.NewPreciseMission(smd.NewEmptySC("prj0", 0), estOrbit, startDT, startDT.Add(-1), estPerts, timeStep, true, smd.ExportConfig{Filename: "prj0", Cosmo: true})
	mEst.RegisterStateChan(stateEstChan)

	// Go-routine to advance propagation.
	go mEst.PropagateUntil(endDT, true)

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
	var covarDistance = 100.0
	var covarVelocity = 0.1
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
	bPlanes := make([]smd.BPlane, 4)
	bPlaneIdx := 0
	for {
		state, more := <-stateEstChan
		if !more {
			break
		}
		stateNo++
		roundedDT := state.DT.Truncate(time.Second)
		switch roundedDT.Sub(firstDT).Hours() {
		case 50 * 24:
			fallthrough
		case 100 * 24:
			fallthrough
		case 150 * 24:
			fallthrough
		case 190 * 24:
			bPlanes[bPlaneIdx] = smd.NewBPlane(state.Orbit)
			bPlaneIdx++
		}
		measurement, exists := measurements[roundedDT]
		if !exists {
			if measNo == 0 {
				time.Sleep(time.Second)
				panic(fmt.Errorf("should start KF at first measurement: \n%s (got)\n%s (exp)", roundedDT, startDT))
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
			estChan <- est
			continue
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
			fmt.Printf("[info] #%04d in visibility of %s (T+%s)\n", measNo, measurement.Station.Name, measurement.State.DT.Sub(startDT))
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
			estChan <- est
		}
		prevDT = measurement.State.DT

		// If in EKF, update the reference trajectory.
		if kf.EKFEnabled() {
			// Update the state from the error.
			R, V := state.Orbit.RV()
			if *debug {
				fmt.Printf("[ekf-] (%04d) %+v\n", measNo, mat64.Formatted(state.Vector().T()))
			}
			for i := 0; i < 3; i++ {
				R[i] += est.State().At(i, 0)
				V[i] += est.State().At(i+3, 0)
			}
			if *debug {
				vec := mat64.NewVector(6, nil)
				for i := 0; i < 3; i++ {
					vec.SetVec(i, R[i])
					vec.SetVec(i+3, V[i])
				}
				fmt.Printf("[ekf+] (%04d) %+v\n", measNo, mat64.Formatted(vec.T()))
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

	// Write BPlane
	f, ferr = os.Create("./bplane.csv")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()
	f.WriteString("bR,bT\n")
	for _, bPlane := range bPlanes {
		csv := fmt.Sprintf("%f,%f\n", bPlane.BR, bPlane.BT)
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
