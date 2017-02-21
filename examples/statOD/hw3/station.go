package main

/*
WARNING: Until issue #67 is done, this is a copy from HW2.
*/

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

// Station defines a ground station.
type Station struct {
	name              string
	R, V              []float64 // position and velocity in ECEF
	latΦ, longθ       float64   // these are stored in radians!
	altitude          float64
	ρNoise, ρDotNoise *distmv.Normal // Station noise
}

// PerformMeasurement returns whether the SC is visible, and if so, the measurement.
func (s Station) PerformMeasurement(θgst float64, state smd.MissionState) (bool, Measurement) {
	// The station vectors are in ECEF, so let's convert the state to ECEF.
	rECEF := smd.ECI2ECEF(state.Orbit.R(), θgst)
	vECEF := smd.ECI2ECEF(state.Orbit.V(), θgst)
	// Compute visibility for each station.

	ρECEF, ρ, el, _ := s.RangeElAz(rECEF)
	if el >= 10 {
		vDiffECEF := make([]float64, 3)
		for i := 0; i < 3; i++ {
			vDiffECEF[i] = (vECEF[i] - s.V[i]) / ρ
		}
		// SC is visible.
		ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
		ρNoisy := ρ + s.ρNoise.Rand(nil)[0]
		ρDotNoisy := ρDot + s.ρDotNoise.Rand(nil)[0]
		// Add this to the list of measurements
		return true, Measurement{ρNoisy, ρDotNoisy, ρ, ρDot, θgst, state, s}
	}
	return false, Measurement{}
}

// RangeElAz returns the range (in the SEZ frame), elevation and azimuth (in degrees) of a given R vector in ECEF.
func (s Station) RangeElAz(rECEF []float64) (ρECEF []float64, ρ, el, az float64) {
	ρECEF = make([]float64, 3)
	for i := 0; i < 3; i++ {
		ρECEF[i] = rECEF[i] - s.R[i]
	}
	ρ = norm(ρECEF)
	rSEZ := smd.MxV33(smd.R3(s.longθ), ρECEF)
	rSEZ = smd.MxV33(smd.R2(math.Pi/2-s.latΦ), rSEZ)
	el = math.Asin(rSEZ[2]/ρ) * r2d
	az = (2*math.Pi + math.Atan2(rSEZ[1], -rSEZ[0])) * r2d
	return
}

// NewStation returns a new station. Angles in degrees.
func NewStation(name string, altitude, latΦ, longθ, σρ, σρDot float64) Station {
	R := smd.GEO2ECEF(altitude, latΦ*d2r, longθ*d2r)
	V := cross([]float64{0, 0, smd.EarthRotationRate}, R)
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	ρNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρ}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	ρDotNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρDot}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	return Station{name, R, V, latΦ * d2r, longθ * d2r, altitude, ρNoise, ρDotNoise}
}

// Measurement stores a measurement of a station.
type Measurement struct {
	ρ, ρDot         float64 // Store the range and range rate
	trueρ, trueρDot float64 // Store the true range and range rate
	θgst            float64
	State           smd.MissionState
	Station         Station
}

// StateVector returns the state vector as a mat64.Vector
func (m Measurement) StateVector() *mat64.Vector {
	return mat64.NewVector(2, []float64{m.ρ, m.ρDot})
}

// HTilde returns the H tilde matrix for this given measurement.
func (m Measurement) HTilde(state smd.MissionState, θgst, θdot float64) *mat64.Dense {
	withRotation := true
	theta := θgst
	thetadot := θdot
	xS := m.Station.R[0]
	yS := m.Station.R[1]
	zS := m.Station.R[2]
	xSDot := m.Station.V[0]
	ySDot := m.Station.V[1]
	zSDot := m.Station.V[2]
	R := state.Orbit.R()
	V := state.Orbit.V()
	x := R[0]
	y := R[1]
	z := R[2]
	xDot := V[0]
	yDot := V[1]
	zDot := V[2]
	rho := m.ρ
	rho2 := math.Pow(m.ρ, 2)
	rhodot := m.ρDot

	H := mat64.NewDense(2, 6, nil)
	if withRotation {
		//drho partials
		drhoDx := (x - xS*math.Cos(theta) + yS*math.Sin(theta)) / rho
		drhoDy := (y - yS*math.Cos(theta) - xS*math.Sin(theta)) / rho
		drhoDz := (z - zS*math.Cos(theta)) / rho

		//drhodot partials
		drhodotDx := (xDot+xS*thetadot*math.Sin(theta)+yS*thetadot*math.Cos(theta))/rho - rhodot*(x-xS*math.Cos(theta)+yS*math.Sin(theta))/rho2
		drhodotDy := (yDot+yS*thetadot*math.Sin(theta)-xS*thetadot*math.Cos(theta))/rho - rhodot*(y-yS*math.Cos(theta)-xS*math.Sin(theta))/rho2
		drhodotDz := zDot/rho - rhodot*(z-zS)/rho2
		drhodotDxdot := (x - xS*math.Cos(theta) - yS*math.Sin(theta)) / rho
		drhodotDydot := (y - yS*math.Cos(theta) - xS*math.Sin(theta)) / rho
		drhodotDzdot := (z - zS) / rho
		// \partial \rho / \partial {x,y,z}
		H.Set(0, 0, drhoDx)
		H.Set(0, 1, drhoDy)
		H.Set(0, 2, drhoDz)
		// \partial \dot\rho / \partial {x,y,z}
		H.Set(1, 0, drhodotDx)
		H.Set(1, 1, drhodotDy)
		H.Set(1, 2, drhodotDz)
		H.Set(1, 3, drhodotDxdot)
		H.Set(1, 4, drhodotDydot)
		H.Set(1, 5, drhodotDzdot)
	} else {
		// \partial \rho / \partial {x,y,z}
		H.Set(0, 0, (x-xS)/m.ρ)
		H.Set(0, 1, (y-yS)/m.ρ)
		H.Set(0, 2, (z-zS)/m.ρ)
		// \partial \dot\rho / \partial {x,y,z}
		H.Set(1, 0, (xDot-xSDot)/m.ρ+(m.ρDot/math.Pow(m.ρ, 2))*(x-xS))
		H.Set(1, 1, (yDot-ySDot)/m.ρ+(m.ρDot/math.Pow(m.ρ, 2))*(y-yS))
		H.Set(1, 2, (zDot-zSDot)/m.ρ+(m.ρDot/math.Pow(m.ρ, 2))*(z-zS))
		H.Set(1, 3, (x-xS)/m.ρ)
		H.Set(1, 4, (y-yS)/m.ρ)
		H.Set(1, 5, (z-zS)/m.ρ)
	}
	return H
}

// CSV returns the data as CSV (does *not* include the new line)
func (m Measurement) CSV() string {
	return fmt.Sprintf("%f,%f,%f,%f,", m.trueρ, m.trueρDot, m.ρ, m.ρDot)
}

func (m Measurement) String() string {
	return fmt.Sprintf("%s@%s", m.Station.name, m.State.DT)
}

// Unshamefully copied from smd/math.go
func cross(a, b []float64) []float64 {
	return []float64{a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0]} // Cross product R x V.
}

// norm returns the norm of a given vector which is supposed to be 3x1.
func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}
