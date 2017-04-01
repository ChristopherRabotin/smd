package smd

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

// Station defines a ground station.
type Station struct {
	Name                       string
	R, V                       []float64 // position and velocity in ECEF
	LatΦ, Longθ                float64   // these are stored in radians!
	Altitude, Elevation        float64
	RangeNoise, RangeRateNoise *distmv.Normal // Station noise
}

// PerformMeasurement returns whether the SC is visible, and if so, the measurement.
func (s Station) PerformMeasurement(θgst float64, state State) Measurement {
	// The station vectors are in ECEF, so let's convert the state to ECEF.
	rECEF := ECI2ECEF(state.Orbit.R(), θgst)
	vECEF := ECI2ECEF(state.Orbit.V(), θgst)
	// Compute visibility for each station.
	ρECEF, ρ, el, _ := s.RangeElAz(rECEF)
	vDiffECEF := make([]float64, 3)
	for i := 0; i < 3; i++ {
		vDiffECEF[i] = (vECEF[i] - s.V[i]) / ρ
	}
	ρDot := mat64.Dot(mat64.NewVector(3, ρECEF), mat64.NewVector(3, vDiffECEF))
	ρNoisy := ρ + s.RangeNoise.Rand(nil)[0]
	ρDotNoisy := ρDot + s.RangeRateNoise.Rand(nil)[0]
	return Measurement{el >= s.Elevation, ρNoisy, ρDotNoisy, ρ, ρDot, θgst, state, s}
}

// RangeElAz returns the range (in the SEZ frame), elevation and azimuth (in degrees) of a given R vector in ECEF.
func (s Station) RangeElAz(rECEF []float64) (ρECEF []float64, ρ, el, az float64) {
	ρECEF = make([]float64, 3)
	for i := 0; i < 3; i++ {
		ρECEF[i] = rECEF[i] - s.R[i]
	}
	ρ = Norm(ρECEF)
	rSEZ := MxV33(R3(s.Longθ), ρECEF)
	rSEZ = MxV33(R2(math.Pi/2-s.LatΦ), rSEZ)
	el = math.Asin(rSEZ[2]/ρ) * r2d
	az = (2*math.Pi + math.Atan2(rSEZ[1], -rSEZ[0])) * r2d
	return
}

// NewStation returns a new station. Angles in degrees.
func NewStation(name string, altitude, elevation, latΦ, longθ, σρ, σρDot float64) Station {
	R := GEO2ECEF(altitude, latΦ*d2r, longθ*d2r)
	V := Cross([]float64{0, 0, EarthRotationRate}, R)
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	ρNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρ}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	ρDotNoise, ok := distmv.NewNormal([]float64{0}, mat64.NewSymDense(1, []float64{σρDot}), seed)
	if !ok {
		panic("NOK in Gaussian")
	}
	return Station{name, R, V, latΦ * d2r, longθ * d2r, altitude, elevation, ρNoise, ρDotNoise}
}

// Measurement stores a measurement of a station.
type Measurement struct {
	Visible                  bool    // Stores whether or not the attempted measurement was visible from the station.
	Range, RangeRate         float64 // Store the range and range rate
	TrueRange, TrueRangeRate float64 // Store the true range and range rate
	Timeθgst                 float64
	State                    State
	Station                  Station
}

// IsNil returns the state vector as a mat64.Vector
func (m Measurement) IsNil() bool {
	return m.Range == m.RangeRate && m.RangeRate == 0
}

// StateVector returns the state vector as a mat64.Vector
func (m Measurement) StateVector() *mat64.Vector {
	return mat64.NewVector(2, []float64{m.Range, m.RangeRate})
}

// HTilde returns the H tilde matrix for this given measurement.
func (m Measurement) HTilde() *mat64.Dense {
	stationR := ECEF2ECI(m.Station.R, m.Timeθgst)
	stationV := ECEF2ECI(m.Station.V, m.Timeθgst)
	xS := stationR[0]
	yS := stationR[1]
	zS := stationR[2]
	xSDot := stationV[0]
	ySDot := stationV[1]
	zSDot := stationV[2]
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
	H.Set(0, 0, (x-xS)/m.Range)
	H.Set(0, 1, (y-yS)/m.Range)
	H.Set(0, 2, (z-zS)/m.Range)
	// \partial \dot\rho / \partial {x,y,z}
	H.Set(1, 0, (xDot-xSDot)/m.Range+(m.RangeRate/math.Pow(m.Range, 2))*(x-xS))
	H.Set(1, 1, (yDot-ySDot)/m.Range+(m.RangeRate/math.Pow(m.Range, 2))*(y-yS))
	H.Set(1, 2, (zDot-zSDot)/m.Range+(m.RangeRate/math.Pow(m.Range, 2))*(z-zS))
	H.Set(1, 3, (x-xS)/m.Range)
	H.Set(1, 4, (y-yS)/m.Range)
	H.Set(1, 5, (z-zS)/m.Range)
	return H
}

// CSV returns the data as CSV (does *not* include the new line)
func (m Measurement) CSV() string {
	return fmt.Sprintf("%f,%f,%f,%f,", m.TrueRange, m.TrueRangeRate, m.Range, m.RangeRate)
}

func (m Measurement) String() string {
	return fmt.Sprintf("%s@%s", m.Station.Name, m.State.DT)
}
