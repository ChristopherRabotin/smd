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

	// Initialize the KF
	Q := mat64.NewSymDense(6, nil)
	R := mat64.NewSymDense(2, []float64{σρ, 0, σρDot, 0})
	noiseKF := gokalman.NewNoiseless(Q, R)

	// Take care of measurements.
	vanillaEstChan := make(chan (gokalman.Estimate), 1)
	go processEst("vanilla", vanillaEstChan)

	prevXHat := mat64.NewVector(6, nil)
	prevP := mat64.NewSymDense(6, nil)
	covarDistance := 10000.
	covarVelocity := 200.
	for i := 0; i < 3; i++ {
		prevP.SetSym(i, i, covarDistance)
		prevP.SetSym(i+3, i+3, covarVelocity)
	}
	var prevθ float64
	var orbit smd.Orbit

	visibilityErrors := 0

	for i, measurement := range measurements {
		fmt.Printf("#%d (%s)\n", i, measurement.Station.name)

		if i == 0 {
			orbit = measurement.State.Orbit
			R, V := orbit.RV()
			for j := 0; j < 3; j++ {
				prevXHat.SetVec(j, R[j])
				prevXHat.SetVec(j+3, V[j])
			}
		}
		// Propagate the reference trajectory until the next measurement time.
		orbitEstimate := smd.NewOrbitEstimate("estimator", orbit, estPerts, measurement.State.DT.Add(-time.Duration(10)*time.Second), time.Second)
		orbitEstimate.PropagateUntil(measurement.State.DT) // This leads to Φ(ti, ti-1)
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
		Φ := orbitEstimate.Φ

		xBar := mat64.NewVector(6, nil)
		xBar.MulVec(Φ, prevXHat)

		PΦ := mat64.NewDense(6, 6, nil)
		PiBar := mat64.NewDense(6, 6, nil)
		PΦ.Mul(prevP, Φ.T())
		PiBar.Mul(Φ, PΦ) // ΦPΦ
		PiBarSym, _ := gokalman.AsSymDense(PiBar)

		// Start the KF now
		vkf, _, _ := gokalman.NewVanilla(prevXHat, PiBarSym, Φ, mat64.NewDense(2, 2, nil), H, noiseKF)
		vest, err := vkf.Update(&y, mat64.NewVector(2, nil))
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		prevXHat = vest.State()
		// Compute residual
		residual := mat64.NewVector(2, nil)
		residual.MulVec(H, vest.State())
		residual.AddScaledVec(residual, -1, &y)
		residual.ScaleVec(-1, residual)
		fmt.Printf("XHat = %+v\n", mat64.Formatted(prevXHat.T()))
		fmt.Printf("residual = %+v\n", mat64.Formatted(residual.T()))

		// Stream to CSV file
		vanillaEstChan <- vest

	}
	close(vanillaEstChan)
	wg.Wait()

	fmt.Printf("\n%d visibility errors\n", visibilityErrors)

}

func processEst(fn string, estChan chan (gokalman.Estimate)) {
	wg.Add(1)
	ce, _ := gokalman.NewCSVExporter([]string{"x", "y", "z", "xDot", "yDot", "zDot"}, ".", fn+".csv")
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
