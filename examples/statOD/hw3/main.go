package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ChristopherRabotin/gokalman"
	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
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
	st1 := NewStation("st1", 0, -35.398333, 148.981944)
	st2 := NewStation("st2", 0, 40.427222, 355.749444)
	st3 := NewStation("st3", 0, 35.247164, 243.205)
	stations := []Station{st1, st2, st3}

	// Noise generation
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	σρ := 1e-3 // m , but all measurements in km.
	ρNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρ}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	σρDot := 1e-6 // mm/s , but all measurements in km/s.
	ρDotNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρDot}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}

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
		str := fmt.Sprintf("%f,", state.DT.Sub(startDT).Seconds())
		θgst := state.DT.Sub(startDT).Seconds() * smd.EarthRotationRate
		// The station vectors are in ECEF, so let's convert the state to ECEF.
		rECEF := smd.ECI2ECEF(state.Orbit.R(), θgst)
		vECEF := smd.ECI2ECEF(state.Orbit.V(), θgst)
		// Compute visibility for each station.
		for _, st := range stations {
			ρECEF, ρ, el, _ := st.RangeElAz(rECEF)
			if el >= 10 {
				vDiffECEF := make([]float64, 3)
				for i := 0; i < 3; i++ {
					vDiffECEF[i] = (vECEF[i] - st.V[i]) / ρ
				}
				// SC is visible.
				ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
				ρNoisy := ρ + ρNoise.Rand(nil)[0]
				ρDotNoisy := ρDot + ρDotNoise.Rand(nil)[0]
				str += fmt.Sprintf("%f,%f,%f,%f,", ρ, ρDot, ρNoisy, ρDotNoisy)
				// Add this to the list of measurements
				measurements = append(measurements, Measurement{ρ, ρDot, θgst, state, st})
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
	perts := smd.Perturbations{Jn: 2}

	// Initialize the KF
	Q := mat64.NewSymDense(6, nil)
	R := mat64.NewSymDense(2, nil)
	noiseKF := gokalman.NewNoiseless(Q, R)

	// Take care of measurements.
	var prevState smd.MissionState
	vanillaEstChan := make(chan (gokalman.Estimate), 1)
	go processEst("vanilla", vanillaEstChan)

	var prevX *mat64.Vector
	var prevP *mat64.SymDense
	//var prevΦ *mat64.Dense
	prevΦ := gokalman.DenseIdentity(6)

	for i, measurement := range measurements {
		fmt.Printf("%d@%s\n", i, measurement.Station.name)
		if i == 0 {
			prevState = measurement.State
			R, V := measurement.State.Orbit.RV()
			prevX = mat64.NewVector(6, []float64{R[0], R[1], R[2], V[0], V[1], V[2]})
			prevP = mat64.NewSymDense(6, nil)
			covarDistance := 100.
			covarVelocity := 10.
			for i := 0; i < 3; i++ {
				prevP.SetSym(i, i, covarDistance)
				prevP.SetSym(i+3, i+3, covarVelocity)
			}
			//continue
		}
		orbitEstimate := smd.NewOrbitEstimate(fmt.Sprintf("est-%d", i), prevState.Orbit, perts, prevState.DT, measurement.State.DT.Sub(prevState.DT), time.Second)
		orbitEstimate.Propagate()
		prevState = orbitEstimate.State()
		var prevΦInv mat64.Dense
		if ierr := prevΦInv.Inverse(prevΦ); ierr != nil {
			panic(fmt.Errorf("could not invert `prevΦ`: %s", ierr))
		}
		var ΦSt mat64.Dense
		ΦSt.Mul(&prevΦInv, orbitEstimate.Φ)
		prevΦ = orbitEstimate.Φ
		Φ := &ΦSt

		xBar := mat64.NewVector(6, nil)
		xBar.MulVec(Φ, prevX)

		rP, cP := prevP.Dims()
		_, cΦ := Φ.Dims()
		PΦ := mat64.NewDense(rP, cΦ, nil)
		PiBar := mat64.NewDense(rP, cP, nil)
		PΦ.Mul(prevP, Φ.T())
		PiBar.Mul(Φ, PΦ) // ΦPΦ
		//fmt.Printf("%+v\n", mat64.Formatted(PiBar))

		// Compute the gain.
		var PHt, HPHt, Ki mat64.Dense
		H := measurement.HTilde()
		PHt.Mul(PiBar, H.T())
		HPHt.Mul(H, &PHt)
		HPHt.Add(&HPHt, noiseKF.MeasurementMatrix())
		fmt.Printf("%+v\n", mat64.Formatted(&HPHt))
		if ierr := HPHt.Inverse(&HPHt); ierr != nil {
			panic(fmt.Errorf("could not invert `H*P_kp1_minus*H' + R`: %s", ierr))
		}
		Ki.Mul(&PHt, &HPHt)

		// Measurement update
		var innov, xHat, xHat1, xHat2 mat64.Vector
		xHat1.MulVec(H, xBar) // Predicted measurement
		// Suppose y_i is nil for now...
		//innov.SubVec(mat64.NewVector(2, nil), mat64.NewVector(2, nil)) // Innovation vector
		innov.SubVec(mat64.NewVector(2, nil), &xHat1) // Innovation vector
		//innov.SubVec(measurement.StateVector(), &xHat1) // Innovation vector
		if rX, _ := innov.Dims(); rX == 1 {
			// xHat1 is a scalar and mat64 won't be happy, so fiddle around to get a vector.
			var sKi mat64.Dense
			sKi.Scale(innov.At(0, 0), &Ki)
			rGain, _ := sKi.Dims()
			xHat2.AddVec(sKi.ColView(0), mat64.NewVector(rGain, nil))
		} else {
			xHat2.MulVec(&Ki, &innov)
		}
		xHat.AddVec(xBar, &xHat2)
		xHat.AddVec(&xHat, noiseKF.Process(0))
		prevX = &xHat

		var PiDense, PiDense1, KiH, KiR, KiRKi mat64.Dense
		KiH.Mul(&Ki, H)
		n, _ := KiH.Dims()
		KiH.Sub(gokalman.Identity(n), &KiH)
		PiDense1.Mul(&KiH, PiBar)
		PiDense.Mul(&PiDense1, KiH.T())
		KiR.Mul(&Ki, noiseKF.MeasurementMatrix())
		KiRKi.Mul(&KiR, Ki.T())
		PiDense.Add(&PiDense, &KiRKi)

		/*PiBarSym, err := gokalman.AsSymDense(PiBar)
		if err != nil {
			panic(err)
		}*/
		PiDenseSym, err := gokalman.AsSymDense(&PiDense)
		if err != nil {
			panic(err)
		}
		prevP = PiDenseSym

		//vanillaEstChan <- gokalman.VanillaEstimate{state: prevX, meas: &ykHat, innovation: &innov, covar: prevP, predCovar: PiBarSym, gain: &Ki}

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
