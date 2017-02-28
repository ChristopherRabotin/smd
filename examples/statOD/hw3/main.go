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

const (
	useEKF = true
)

var wg sync.WaitGroup

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
		fmt.Println(hdr)
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
	if useEKF {
		go processEst("extended", estChan)
	} else {
		go processEst("vanilla", estChan)
	}

	prevXHat := mat64.NewVector(6, nil)
	prevP := mat64.NewSymDense(6, nil)
	covarDistance := 10.
	covarVelocity := 2.
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}
	var prevθ float64
	var prevΦ *mat64.Dense
	var orbit smd.Orbit

	visibilityErrors := 0
	var orbitEstimate *smd.OrbitEstimate

	for i, measurement := range measurements {
		fmt.Printf("#%d (%s)\n", i, measurement.Station.name)

		if i == 0 {
			orbit = measurement.State.Orbit
			R, V := orbit.RV()
			for j := 0; j < 3; j++ {
				prevXHat.SetVec(j, R[j])
				prevXHat.SetVec(j+3, V[j])
			}
			orbitEstimate = smd.NewOrbitEstimate("estimator", orbit, estPerts, measurement.State.DT.Add(-time.Duration(10)*time.Second), time.Second)
		}
		var Φ mat64.Dense
		if useEKF {
			// Only use actual EKF after a few iterations.
			// Generate the new orbit estimate from the previous estimated state error.
			if i > 10 {
				// We just computed this for i==0.
				// In the case of the EKF, the prevXHat is the difference between the reference trajectory and the real one.
				// So let's recreate an orbit.
				R, V := orbit.RV()
				for k := 0; k < 3; k++ {
					R[k] = prevXHat.At(k, 0)
					V[k] = prevXHat.At(k+3, 0)
				}
				orbit = *smd.NewOrbitFromRV(R, V, smd.Earth)
				fmt.Printf("%s\n", orbit)
				orbitEstimate = smd.NewOrbitEstimate("estimator", orbit, estPerts, measurement.State.DT.Add(-time.Duration(10)*time.Second), time.Second)
				// Propagate the reference trajectory until the next measurement time.
				orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(ti, ti-1) because we are restarting the integration.
			}
			Φ = *orbitEstimate.Φ
		} else {
			prevΦ = orbitEstimate.Φ // Store the previous estimate of Phi before propagation. (which is Φ(t0, ti-1))
			// Propagate the reference trajectory until the next measurement time.
			orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(t0, ti)

			var invΦ mat64.Dense
			if ierr := invΦ.Inverse(prevΦ); ierr != nil {
				panic(fmt.Errorf("could not invert `est1.Φ`: %s", ierr))
			}
			// Now we have Φ(ti-1, t0)
			Φ.Mul(orbitEstimate.Φ, &invΦ)
		}
		// Compute "real" measurement
		vis, expMeas := measurement.Station.PerformMeasurement(measurement.θgst, orbitEstimate.State())
		if !vis {
			fmt.Printf("[warning] station %s should see the SC but does not\n", measurement.Station.name)
			visibilityErrors++
		}
		var y mat64.Vector
		y.SubVec(measurement.StateVector(), expMeas.StateVector())
		// Compute H tilde
		θdot := measurement.θgst - prevθ
		H := measurement.HTilde(orbitEstimate.State(), measurement.θgst, θdot)

		xBar := mat64.NewVector(6, nil)
		xBar.MulVec(&Φ, prevXHat)

		PΦ := mat64.NewDense(6, 6, nil)
		PiBar := mat64.NewDense(6, 6, nil)
		PΦ.Mul(prevP, Φ.T())
		PiBar.Mul(&Φ, PΦ) // ΦPΦ
		PiBarSym, _ := gokalman.AsSymDense(PiBar)

		// Start the KF now
		var kf gokalman.KalmanFilter
		if useEKF {
			// the x0 in the extended KF is not used.
			kf, _, _ = gokalman.NewExtended(mat64.NewVector(6, nil), PiBarSym, &Φ, mat64.NewDense(2, 2, nil), H, noiseKF)
		} else {
			kf, _, _ = gokalman.NewVanilla(prevXHat, PiBarSym, &Φ, mat64.NewDense(2, 2, nil), H, noiseKF)
		}
		est, err := kf.Update(&y, mat64.NewVector(2, nil))
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		prevXHat = est.State()
		prevP = est.Covariance().(*mat64.SymDense)
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(H, est.State())
		residual.AddScaledVec(residual, -1, &y)
		residual.ScaleVec(-1, residual)
		residuals[i] = residual

		// Stream to CSV file
		estChan <- truth.Error(i, est)

	}
	close(estChan)
	wg.Wait()

	fmt.Printf("\n%d visibility errors\n", visibilityErrors)
	// Write the residuals to a CSV file
	fname := "ekf"
	if !useEKF {
		fname = "vkf"
	}
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
