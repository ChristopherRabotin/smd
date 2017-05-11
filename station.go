package smd

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/gonum/matrix/mat64"
	"github.com/gonum/stat/distmv"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

var (
	σρ             = math.Pow(5e-3, 2) // m , but all measurements in km.
	σρDot          = math.Pow(5e-6, 2) // m/s , but all measurements in km/s.
	DSS34Canberra  = NewSpecialStation("DSS34Canberra", 0.691750, 0, -35.398333, 148.981944, σρ, σρDot, 6)
	DSS65Madrid    = NewSpecialStation("DSS65Madrid", 0.834939, 0, 40.427222, 4.250556, σρ, σρDot, 6)
	DSS13Goldstone = NewSpecialStation("DSS13Goldstone", 1.07114904, 0, 35.247164, 243.205, σρ, σρDot, 6)
)

// Station defines a ground station.
type Station struct {
	Name                       string
	R, V                       []float64 // position and velocity in ECEF
	LatΦ, Longθ                float64   // these are stored in radians!
	Altitude, Elevation        float64
	RangeNoise, RangeRateNoise *distmv.Normal // Station noise
	Planet                     CelestialObject
	rowsH                      int // If estimating Cr in addition to position and velocity, this needs to be 7
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

func (s Station) String() string {
	return fmt.Sprintf("%s (%f,%f); alt = %f km; el = %f deg", s.Name, s.LatΦ/d2r, s.Longθ/d2r, s.Altitude, s.Elevation)
}

// NewStation returns a new station. Angles in degrees.
func NewStation(name string, altitude, elevation, latΦ, longθ, σρ, σρDot float64) Station {
	return NewSpecialStation(name, altitude, elevation, latΦ, longθ, σρ, σρDot, 6)
}

// NewSpecialStation same as NewStation but can specify the rows of H.
func NewSpecialStation(name string, altitude, elevation, latΦ, longθ, σρ, σρDot float64, rowsH int) Station {
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
	return Station{name, R, V, latΦ * d2r, longθ * d2r, altitude, elevation, ρNoise, ρDotNoise, Earth, rowsH}
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
	H := mat64.NewDense(2, m.Station.rowsH, nil)
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

// ShortCSV returns the noisy data as CSV (does *not* include the new line)
func (m Measurement) ShortCSV() string {
	return fmt.Sprintf("%f,%f,", m.Range, m.RangeRate)
}

func (m Measurement) String() string {
	return fmt.Sprintf("%s@%s", m.Station.Name, m.State.DT)
}

func BuiltinStationFromName(name string) Station {
	switch strings.ToLower(name) {
	case "dss13":
		return DSS13Goldstone
	case "dss34":
		return DSS34Canberra
	case "dss65":
		return DSS65Madrid
	default:
		panic(fmt.Errorf("unknown station `%s`", name))
	}
}
