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
	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
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
	vDiffECEF := make([]float64, 3)
	for i := 0; i < 3; i++ {
		vDiffECEF[i] = (vECEF[i] - s.V[i]) / ρ
	}
	// Suppose SC is visible.
	ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
	ρNoisy := ρ + s.ρNoise.Rand(nil)[0]
	ρDotNoisy := ρDot + s.ρDotNoise.Rand(nil)[0]
	// Add this to the list of measurements
	// TODO: Change signature
	return el >= 10, Measurement{el >= 10, ρNoisy, ρDotNoisy, ρ, ρDot, θgst, state, s}
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
	Visible         bool    // Stores whether or not the attempted measurement was visible from the station.
	ρ, ρDot         float64 // Store the range and range rate
	trueρ, trueρDot float64 // Store the true range and range rate
	θgst            float64
	State           smd.MissionState
	Station         Station
}

// IsNil returns the state vector as a mat64.Vector
func (m Measurement) IsNil() bool {
	return m.ρ == m.ρDot && m.ρDot == 0
}

// StateVector returns the state vector as a mat64.Vector
func (m Measurement) StateVector() *mat64.Vector {
	return mat64.NewVector(2, []float64{m.ρ, m.ρDot})
}

// HTilde returns the H tilde matrix for this given measurement.
func (m Measurement) HTilde(state smd.MissionState, θgst float64) *mat64.Dense {
	stationR := smd.ECEF2ECI(m.Station.R, θgst)
	stationV := smd.ECEF2ECI(m.Station.V, θgst)
	xS := stationR[0]
	yS := stationR[1]
	zS := stationR[2]
	xSDot := stationV[0]
	ySDot := stationV[1]
	zSDot := stationV[2]
	R := state.Orbit.R()
	V := state.Orbit.V()
	x := R[0]
	y := R[1]
	z := R[2]
	xDot := V[0]
	yDot := V[1]
	zDot := V[2]
	H := mat64.NewDense(2, 6, nil)
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

// unit returns the unit vector of a given vector.
func unit(a []float64) (b []float64) {
	n := norm(a)
	if floats.EqualWithinAbs(n, 0, 1e-12) {
		return []float64{0, 0, 0}
	}
	b = make([]float64, len(a))
	for i, val := range a {
		b[i] = val / n
	}
	return
}
