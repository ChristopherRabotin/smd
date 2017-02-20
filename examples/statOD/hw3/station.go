package main

/*
WARNING: Until issue #67 is done, this is a copy from HW2.
*/

import (
	"fmt"
	"math"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

// Station defines a ground station.
type Station struct {
	name        string
	R, V        []float64 // position and velocity in ECEF
	latΦ, longθ float64   // these are stored in radians!
	altitude    float64
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
func NewStation(name string, altitude, latΦ, longθ float64) Station {
	R := smd.GEO2ECEF(altitude, latΦ*d2r, longθ*d2r)
	V := cross([]float64{0, 0, smd.EarthRotationRate}, R)
	return Station{name, R, V, latΦ * d2r, longθ * d2r, altitude}
}

// Measurement stores a measurement of a station.
type Measurement struct {
	ρ, ρDot float64 // Store the range and range rate
	θgst    float64
	State   smd.MissionState
	Station Station
}

// StateVector returns the state vector as a mat64.Vector
func (m Measurement) StateVector() *mat64.Vector {
	return mat64.NewVector(2, []float64{m.ρ, m.ρDot})
}

// HTilde returns the H tilde matrix for this given measurement.
func (m Measurement) HTilde() *mat64.Dense {
	xS := m.Station.R[0]
	yS := m.Station.R[1]
	zS := m.Station.R[2]
	xSDot := m.Station.V[0]
	ySDot := m.Station.V[1]
	zSDot := m.Station.V[2]
	// Compute the R and V in ECEF
	//R := smd.ECI2ECEF(m.State.Orbit.R(), m.θgst)
	//V := smd.ECI2ECEF(m.State.Orbit.V(), m.θgst)
	R := m.State.Orbit.R()
	V := m.State.Orbit.V()
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
