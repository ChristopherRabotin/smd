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

// Scenario constants
const (
	initJDE = 2456296.25
	realBT  = 7009.767
	realBR  = 140002.894
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
	timeStep        = 30 * time.Second
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

	// Compute number of states which will be generated.
	numStates := int(endDT.Sub(startDT).Seconds()/timeStep.Seconds()) + 1
	residuals := make([]*mat64.Vector, numStates+1)

	// Get the first measurement as an initial orbit estimation.
	firstDT := startDT
	estOrbit := smd.NewOrbitFromRV([]float64{-274096790.0, -92859240.0, -40199490.0}, []float64{32.67, -8.94, -3.88}, smd.Earth)
	startDT = firstDT.Add(-timeStep)
	// Perturbations in the estimate
	estPerts := smd.Perturbations{PerturbingBody: &smd.Sun, Drag: true}

	stateEstChan := make(chan (smd.State))
	sc := smd.NewEmptySC("prj0", 0)
	sc.Drag = 1.2
	mEst := smd.NewPreciseMission(sc, estOrbit, startDT, startDT.Add(-1), estPerts, timeStep, true, smd.ExportConfig{Filename: "prj0", Cosmo: true})
	mEst.RegisterStateChan(stateEstChan)

	// Go-routine to advance propagation.
	go mEst.PropagateUntil(endDT, true)

	// KF filter initialization stuff.

	// Initialize the KF noise
	σQx := math.Pow(10, -2*σQExponent)
	var σQy, σQz float64
	noiseQ := mat64.NewSymDense(3, []float64{σQx, 0, 0, 0, σQy, 0, 0, 0, σQz})
	noiseR := mat64.NewSymDense(2, []float64{σρ, 0, 0, σρDot})
	noiseKF := gokalman.NewNoiseless(noiseQ, noiseR)

	// Take care of measurements.
	estChan := make(chan (gokalman.Estimate), 1)
	filename := fmt.Sprintf("srif-part-%s", measFile)
	go processEst(filename, estChan)

	prevP := mat64.NewSymDense(7, nil)
	var covarDistance = 100.0
	var covarVelocity = 0.1
	prevP.SetSym(6, 6, 0.1)
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}

	visibilityErrors := 0

	var prevStationName = ""
	measNo := 0
	stateNo := 0
	kf, _, err := gokalman.NewSRIF(mat64.NewVector(7, nil), prevP, 2, false, noiseKF)
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
				panic(fmt.Errorf("should start KF at first measurement: \n%s (got)\n%s (exp)", roundedDT, startDT))
			}
			// There is no truth measurement here, let's only predict the KF covariance.
			kf.Prepare(state.Φ, nil)
			est, perr := kf.Predict()
			if perr != nil {
				panic(fmt.Errorf("[ERR!] (#%04d)\n%s", measNo, perr))
			}
			stateEst := mat64.NewVector(7, nil)
			stateEst.SubVec(est.State(), state.Vector())
			// NOTE: The state seems to be all I need, along with Phi maybe (?) because the KF already uses the previous state?!
			if *debug {
				fmt.Printf("[pred] (%04d) %+v\n", measNo, mat64.Formatted(est.State().T()))
			}
			estChan <- est
			continue
		}

		// Let's perform a full update since there is a measurement.
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
		est, err := kf.Update(measurement.StateVector(), computedObservation.StateVector())
		if err != nil {
			panic(fmt.Errorf("[ERR!] %s", err))
		}

		prevP = est.Covariance().(*mat64.SymDense)
		stateEst := mat64.NewVector(7, nil)
		stateEst.AddVec(state.Vector(), est.State())
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(Htilde, est.State())
		residual.AddScaledVec(residual, -1, est.ObservationDev())
		residual.ScaleVec(-1, residual)
		residuals[stateNo] = residual
		// Stream to CSV file
		estChan <- est

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
	f, ferr := os.Create(fmt.Sprintf("./%s-residuals.csv", filename))
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
	f.WriteString(fmt.Sprintf("%f,%f\n", realBR, realBT))
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
