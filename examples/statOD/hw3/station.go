package main

/*
WARNING: Until issue #67 is done, this is a copy from HW2.
*/

import (
	"math"

	"github.com/ChristopherRabotin/smd"
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
