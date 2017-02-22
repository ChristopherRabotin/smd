package main

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
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

	// Perturbations in the estimate
	estPerts := smd.Perturbations{Jn: 2}
	// Start estimate at an initial reference trajectory
	var orbitEstimate *smd.OrbitEstimate

	// Initialize the KF
	Q := mat64.NewSymDense(6, nil)
	R := mat64.NewSymDense(2, []float64{σρ, 0, σρDot, 0})
	noiseKF := gokalman.NewNoiseless(Q, R)

	// Take care of measurements.
	vanillaEstChan := make(chan (gokalman.Estimate), 1)
	go processEst("vanilla", vanillaEstChan)

	var prevXHat *mat64.Vector
	var prevP *mat64.SymDense
	var prevθ float64

	prevΦ := gokalman.DenseIdentity(6)

	for i, measurement := range measurements {
		fmt.Printf("%d@%s\n", i, measurement.Station.name)
		if i == 0 {
			R, V := measurement.State.Orbit.RV()
			prevXHat = mat64.NewVector(6, []float64{R[0], R[1], R[2], V[0], V[1], V[2]})
			//prevXHat = mat64.NewVector(6, nil)
			prevP = mat64.NewSymDense(6, nil)
			covarDistance := 100.
			covarVelocity := 10.
			for i := 0; i < 3; i++ {
				prevP.SetSym(i, i, covarDistance)
				prevP.SetSym(i+3, i+3, covarVelocity)
			}
			prevθ = measurement.θgst
			// Initialize the orbit estimate at this first measurement
			orbitEstimate = smd.NewOrbitEstimate("estimator", measurement.State.Orbit, estPerts, measurement.State.DT, time.Second)
			continue
		}
		// DEBUG
		R, V := measurement.State.Orbit.RV()
		prevXHat = mat64.NewVector(6, []float64{R[0], R[1], R[2], V[0], V[1], V[2]})
		// END DEBUG
		// Propagate the reference trajectory until the next measurement time.
		orbitEstimate.PropagateUntil(measurement.State.DT)
		var prevΦInv mat64.Dense
		if ierr := prevΦInv.Inverse(prevΦ); ierr != nil {
			panic(fmt.Errorf("could not invert `prevΦ`: %s", ierr))
		}
		//fmt.Printf("prevΦInv\n%+v\n", mat64.Formatted(&prevΦInv))
		var ΦSt mat64.Dense
		ΦSt.Mul(orbitEstimate.Φ, &prevΦInv)
		prevΦ = orbitEstimate.Φ
		Φ := &ΦSt

		xBar := mat64.NewVector(6, nil)
		xBar.MulVec(Φ, prevXHat)

		rP, cP := prevP.Dims()
		_, cΦ := Φ.Dims()
		PΦ := mat64.NewDense(rP, cΦ, nil)
		PiBar := mat64.NewDense(rP, cP, nil)
		PΦ.Mul(prevP, Φ.T())
		PiBar.Mul(Φ, PΦ) // ΦPΦ

		// Compute innovation
		vis, expMeas := measurement.Station.PerformMeasurement(measurement.θgst, orbitEstimate.State())
		if !vis {
			panic(fmt.Errorf("station %s should see the SC but does not", measurement.Station.name))
		}
		var y mat64.Vector
		y.SubVec(measurement.StateVector(), expMeas.StateVector())

		// Compute H tilde
		θdot := measurement.θgst - prevθ
		H := measurement.HTilde(orbitEstimate.State(), measurement.θgst, θdot)

		// Compute the gain.
		var PHt, HPHt, Ki mat64.Dense
		PHt.Mul(PiBar, H.T())
		HPHt.Mul(H, &PHt)
		HPHt.Add(&HPHt, noiseKF.MeasurementMatrix())
		fmt.Printf("Pibar\n%+v\n", mat64.Formatted(PiBar))
		if ierr := HPHt.Inverse(&HPHt); ierr != nil {
			panic(fmt.Errorf("could not invert `H*P_kp1_minus*H' + R`: %s", ierr))
		}
		Ki.Mul(&PHt, &HPHt)

		// Measurement update
		var xHat, xHat1, xHat2 mat64.Vector
		xHat1.MulVec(H, xBar) // Predicted measurement
		xHat1.SubVec(&y, &xHat1)
		xHat2.MulVec(&Ki, &xHat1)
		xHat.AddVec(xBar, &xHat2)
		prevXHat = &xHat

		var PiDense, KiH, KiR, KiRKi mat64.Dense
		KiH.Mul(&Ki, H)
		n, _ := KiH.Dims()
		KiH.Sub(gokalman.Identity(n), &KiH)
		PiDense.Mul(&KiH, PiBar)
		//PiDense.Mul(&PiDense1, KiH.T())
		KiR.Mul(&Ki, noiseKF.MeasurementMatrix())
		KiRKi.Mul(&KiR, Ki.T())
		PiDense.Add(&PiDense, &KiRKi)

		PiDenseSym, err := gokalman.AsSymDense(&PiDense)
		if err != nil {
			panic(err)
		}
		prevP = PiDenseSym

		//vanillaEstChan <- gokalman.VanillaEstimate{state: prevXHat, meas: &ykHat, yation: &y, covar: prevP, predCovar: PiBarSym, gain: &Ki}

	}
	//close(vanillaEstChan)
	//wg.Wait()

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
