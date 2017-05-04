package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/spf13/viper"
)

// Scenario constants
const (
	defaultScenario = "~~unset~~"
	dateFormat      = "2006-01-02 15:04:05"
)

var (
	kf             gokalman.NLDKF
	scenario       string
	wg             sync.WaitGroup
	ekfTrigger     int
	ekfDisableTime float64
	sncEnabled     bool
	sncDisableTime float64
	sncRIC         bool
	smoothing      bool
)

var debug = flag.Bool("debug", false, "verbose debug")

func init() {
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
}

func main() {
	flag.Parse()
	// Load scenario
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}

	scenario = strings.Replace(scenario, ".toml", "", 1)
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("./%s.toml: Error %s", scenario, err)
	}

	// Read Mission parameters
	startDT := confReadJDEorTime("mission.start")
	endDT := confReadJDEorTime("mission.end")
	timeStep := viper.GetDuration("mission.step")

	// Read orbit
	sc := smd.NewEmptySC("fltr", 0)
	var scOrbit *smd.Orbit
	centralBodyName := viper.GetString("orbit.body")
	centralBody, err := smd.CelestialObjectFromString(centralBodyName)
	if err != nil {
		log.Fatalf("could not understand body `%s`: %s", centralBodyName, err)
	}
	if viper.GetBool("viaRV") {
		R := make([]float64, 3)
		V := make([]float64, 3)
		for i := 0; i < 3; i++ {
			R[i] = viper.GetFloat64(fmt.Sprintf("orbit.R%d", i+1))
			V[i] = viper.GetFloat64(fmt.Sprintf("orbit.V%d", i+1))
		}
		scOrbit = smd.NewOrbitFromRV(R, V, centralBody)
	} else {
		a := viper.GetFloat64("orbit.sma")
		e := viper.GetFloat64("orbit.ecc")
		i := viper.GetFloat64("orbit.inc")
		Ω := viper.GetFloat64("orbit.RAAN")
		ω := viper.GetFloat64("orbit.argPeri")
		ν := viper.GetFloat64("orbit.tAnomaly")
		scOrbit = smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, centralBody)
	}

	// Read stations
	stationNames := viper.GetStringSlice("measurements.stations")
	stations := make(map[string]smd.Station)
	for _, stationName := range stationNames {
		var st smd.Station
		if len(stationName) > 8 && stationName[0:8] == "builtin." {
			st = smd.BuiltinStationFromName(stationName[8:len(stationName)])
			stations[st.Name] = st
		} else {
			// Read provided station.
			stationKey := fmt.Sprintf("station.%s.", stationName)
			humanName := viper.GetString(stationKey + "name")
			altitude := viper.GetFloat64(stationKey + "altitude")
			elevation := viper.GetFloat64(stationKey + "elevation")
			latΦ := viper.GetFloat64(stationKey + "latitude")
			longθ := viper.GetFloat64(stationKey + "longitude")
			σρ := viper.GetFloat64(stationKey + "range_sigma")
			σρDot := viper.GetFloat64(stationKey + "rate_sigma")
			st = smd.NewStation(humanName, altitude, elevation, latΦ, longθ, σρ, σρDot)
			if planetName := viper.GetString(stationKey + "planet"); len(planetName) > 0 {
				// A planet was specified, so it might not be Earth
				if planet, errp := smd.CelestialObjectFromString(planetName); errp != nil {
					log.Fatalf("could not use `%s` as planet for station `%s`: %s", planetName, humanName, err)
				} else {
					st.Planet = planet
				}
			}
			stations[humanName] = st
		}
		log.Printf("[info] added station %s", st)
	}

	// Load measurement file
	measurements, measStartDT, measEndDT := loadMeasurementFile(viper.GetString("measurements.file"), stations)
	log.Printf("[info] Loaded %d measurements from %s to %s", len(measurements), measStartDT, measEndDT)

	// Check overlap between measurement file and the dates of the mission.
	if viper.GetBool("mission.autodate") {
		startDT = measStartDT
		endDT = measEndDT
	} else if startDT.After(measEndDT) {
		log.Fatal("mission start time is after last measurement")
	}

	// Read SNC
	sncEnabled = viper.GetBool("SNC.enabled")
	sncRIC = viper.GetBool("SNC.RICframe")
	sncDisableTime = viper.GetFloat64("SNC.disableTime")

	// Read filter configuration
	var fltType gokalman.FilterType
	fltTypeString := viper.GetString("filter.type")
	fltFilePrefix := viper.GetString("filter.outPrefix")
	switch fltTypeString {
	case "EKF":
		fltType = gokalman.EKFType
		ekfDisableTime = viper.GetFloat64("EKF.disableTime")
		ekfTrigger = viper.GetInt("EKF.trigger")
	case "CKF":
		fltType = gokalman.CKFType
		smoothing = viper.GetBool(fmt.Sprintf("%s.smooth", fltTypeString))
	case "SRIF":
		fltType = gokalman.SRIFType
		smoothing = viper.GetBool(fmt.Sprintf("%s.smooth", fltTypeString))
	case "UKF":
		fltType = gokalman.UKFType
		panic("filter UKF not yet implementation")
	default:
		panic(fmt.Errorf("unknown filter `%s`", fltTypeString))
	}

	// Read variance
	σQx := viper.GetFloat64("variance.Q")
	var σQy, σQz float64
	if !sncRIC {
		σQy = σQx
		σQz = σQx
	}
	noiseQ := mat64.NewSymDense(3, []float64{σQx, 0, 0, 0, σQy, 0, 0, 0, σQz})
	noiseR := mat64.NewSymDense(2, []float64{viper.GetFloat64("noise.range"), 0, 0, viper.GetFloat64("noise.rate")})
	noiseKF := gokalman.NewNoiseless(noiseQ, noiseR)

	// Compute number of states which will be generated.
	numStates := int(endDT.Sub(startDT).Seconds()/timeStep.Seconds()) + 1
	residuals := make([]*mat64.Vector, numStates)

	// TODO: Perturbations in the estimate
	estPerts := smd.Perturbations{PerturbingBody: &smd.Sun}

	stateEstChan := make(chan (smd.State), 1)

	mEst := smd.NewPreciseMission(sc, scOrbit, startDT, startDT.Add(-1), estPerts, timeStep, true, smd.ExportConfig{Filename: "prj0", Cosmo: true})
	mEst.RegisterStateChan(stateEstChan)

	// Go-routine to advance propagation.
	go mEst.PropagateUntil(endDT.Add(timeStep), true)

	// KF filter initialization stuff.

	// Take care of measurements.
	var estHistory []gokalman.Estimate
	estChan := make(chan (gokalman.Estimate), 1)
	go processEst(fltFilePrefix, estChan)

	prevP := mat64.NewSymDense(6, nil)
	var covarDistance = viper.GetFloat64("covariance.position")
	var covarVelocity = viper.GetFloat64("covariance.velocity")
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}

	visibilityErrors := 0

	if smoothing {
		log.Println("[info] Smoothing enabled")
	}

	if fltType == gokalman.EKFType {
		if ekfTrigger < 0 {
			log.Println("[WARNING] EKF disabled")
		} else {
			if smoothing {
				log.Println("[ERROR] Enabling smooth has NO effect because EKF is enabled")
			}
			if ekfTrigger < 10 {
				log.Println("[WARNING] EKF may be turned on too early")
			} else {
				fmt.Printf("[info] EKF will turn on after %d measurements\n", ekfTrigger)
			}
		}
	}

	var prevStationName = ""
	var prevDT time.Time
	var ckfMeasNo = 0
	measNo := 1
	stateNo := 0
	x0 := mat64.NewVector(6, nil)
	if fltType == gokalman.EKFType || fltType == gokalman.CKFType {
		kf, _, err = gokalman.NewHybridKF(x0, prevP, noiseKF, 2)
		if err != nil {
			panic(fmt.Errorf("%s", err))
		}
	} else if fltType == gokalman.SRIFType {
		kf, _, err = gokalman.NewSRIF(x0, prevP, 2, false, noiseKF)
		if err != nil {
			panic(fmt.Errorf("%s", err))
		}
	}

	for state := range stateEstChan {
		stateNo++
		roundedDT := state.DT.Truncate(time.Second)
		measurement, exists := measurements[roundedDT]
		if !exists {
			if measNo == 0 {
				panic(fmt.Errorf("should start KF at first measurement: \n%s (got)\n%s (exp)", roundedDT, startDT))
			}
			// There is no truth measurement here, let's only predict the KF covariance.
			kf.Prepare(state.Φ, nil)
			estI, perr := kf.Predict()
			if perr != nil {
				panic(fmt.Errorf("[ERR!] (#%05d)\n%s", measNo, perr))
			}
			est := estI.(*gokalman.HybridKFEstimate)
			// NOTE: The state seems to be all I need, along with Phi maybe (?) because the KF already uses the previous state?!
			if *debug {
				fmt.Printf("[pred] (%05d) %+v\n", measNo, mat64.Formatted(est.State().T()))
			}
			if smoothing {
				// Save to history in order to perform smoothing.
				estHistory[stateNo-1] = estI
			} else {
				// Stream to CSV file
				estChan <- est
			}
			continue
		}

		if measNo == 0 {
			prevDT = measurement.State.DT
		}

		// Let's perform a full update since there is a measurement.
		ΔtDuration := measurement.State.DT.Sub(prevDT)
		Δt := ΔtDuration.Seconds() // Everything is in seconds.
		// Informational messages.
		if !kf.EKFEnabled() && ckfMeasNo == ekfTrigger {
			// Switch KF to EKF mode
			kf.EnableEKF()
			fmt.Printf("[info] #%05d EKF now enabled\n", measNo)
		} else if kf.EKFEnabled() && ekfDisableTime > 0 && Δt > ekfDisableTime {
			// Switch KF back to CKF mode
			kf.DisableEKF()
			ckfMeasNo = 0
			fmt.Printf("[info] #%05d EKF now disabled (Δt=%s)\n", measNo, ΔtDuration)
		}

		if measurement.Station.Name != prevStationName {
			fmt.Printf("[info] #%05d %s in visibility (T+%s)\n", measNo, measurement.Station.Name, measurement.State.DT.Sub(startDT))
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
		estI, err := kf.Update(measurement.StateVector(), computedObservation.StateVector())
		if err != nil {
			panic(fmt.Errorf("[ERR!] %s", err))
		}
		est := estI.(*gokalman.HybridKFEstimate)
		prevP = est.Covariance().(*mat64.SymDense)

		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(Htilde, est.State())
		residual.AddScaledVec(residual, -1, est.ObservationDev())
		residual.ScaleVec(-1, residual)
		residuals[stateNo-1] = residual

		prevDT = measurement.State.DT
		// Stream to CSV file
		if smoothing {
			// Save to history in order to perform smoothing.
			estHistory[stateNo-1] = est
		} else {
			// Stream to CSV file
			estChan <- est
		}
		// If in EKF, update the reference trajectory.
		if kf.EKFEnabled() {
			// Update the state from the error.
			R, V := state.Orbit.RV()
			if *debug {
				log.Printf("[ekf-] (%04d) %+v\n", measNo, mat64.Formatted(state.Vector().T()))
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
				log.Printf("[ekf+] (%04d) %+v\n", measNo, mat64.Formatted(vec.T()))
			}
			mEst.Orbit = smd.NewOrbitFromRV(R, V, smd.Earth)
		}
		ckfMeasNo++
		measNo++
	} // end while true

	if smoothing {
		log.Println("[info] Smoothing started")
		// Perform the smoothing. First, play back all the estimates backward, and then replay the smoothed estimates forward to compute the difference.
		// Cast the filter into what is selected.
		if fltType == gokalman.SRIFType {
			// Create another list of history for smoothing (cannot cast slice)
			estHistoryClone := make([]*gokalman.SRIFEstimate, numStates)
			for i := 0; i < numStates; i++ {
				estHistoryClone[i] = estHistory[i].(*gokalman.SRIFEstimate)
			}
			if err := kf.(*gokalman.SRIF).SmoothAll(estHistoryClone); err != nil {
				panic(err)
			}
		} else {
			// Create another list of history for smoothing (cannot cast slice)
			estHistoryClone := make([]*gokalman.HybridKFEstimate, numStates)
			for i := 0; i < numStates; i++ {
				estHistoryClone[i] = estHistory[i].(*gokalman.HybridKFEstimate)
			}
			if err := kf.(*gokalman.HybridKF).SmoothAll(estHistoryClone); err != nil {
				panic(err)
			}
		}
		// Replay forward
		for _, estimate := range estHistory {
			estChan <- estimate
		}
		log.Println("[info] Smoothing completed")
	}

	close(estChan)
	wg.Wait()

	severity := "info"
	if visibilityErrors > 0 {
		severity = "WARNING"
	}
	log.Printf("[%s] %d visibility errors\n", severity, visibilityErrors)
	// Write the residuals to a CSV file
	f, ferr := os.Create(fmt.Sprintf("%s-residuals.csv", fltFilePrefix))
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
	// We also compute the RMS here and write the pre-fit residuals.
	// Write BPlane
	f, ferr := os.Create(fmt.Sprintf("./%s-prefit.csv", fn))
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()
	f.WriteString("rho,rhoDot\n")
	numMeasurements := 0
	rmsPosition := 0.0
	rmsVelocity := 0.0
	ce, _ := gokalman.NewCustomCSVExporter([]string{"x", "y", "z", "xDot", "yDot", "zDot", "Cr"}, ".", fn+".csv", 3)
	for {
		est, more := <-estChan
		if !more {
			ce.Close()
			wg.Done()
			break
		}
		f.WriteString(fmt.Sprintf("%f,%f\n", est.Innovation().At(0, 0), est.Innovation().At(1, 0)))
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
