package main

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
}

// RangeElAz returns the range (in the SEZ frame), elevation and azimuth (in degrees) of a given R vector in ECEF.
func (s Station) RangeElAz(R []float64) (ρECEF []float64, ρ, el, az float64) {
	ρECEF = make([]float64, 3)
	for i := 0; i < 3; i++ {
		ρECEF[i] = R[i] - s.R[i]
	}
	r2r3 := mat64.NewDense(3, 3, nil)
	r2r3.Mul(smd.R3(s.longθ), smd.R2(math.Pi/2-s.latΦ))
	ρSEZ := mat64.NewVector(3, nil)
	ρSEZ.MulVec(r2r3, mat64.NewVector(3, ρECEF))
	ρ = mat64.Norm(ρSEZ, 2)
	el = (math.Asin(ρSEZ.At(2, 0) / ρ)) * r2d
	az = (math.Asin(ρSEZ.At(2, 0) / math.Sqrt(math.Pow(ρSEZ.At(0, 0), 2)+math.Pow(ρSEZ.At(1, 0), 2)))) * r2d
	if el >= 10 {
		fmt.Printf("%+v\t%f\t%f\t%f\n", mat64.Formatted(ρSEZ), ρ, mat64.Norm(mat64.NewVector(3, R), 2), el)
	}
	return
}

// NewStation returns a new station. Angles in degrees.
func NewStation(name string, altitude, latΦ, longθ float64) Station {
	R := smd.GEO2ECEF(altitude, latΦ, longθ)
	V := cross([]float64{0, 0, smd.EarthRotationRate}, R)
	return Station{name, R, V, latΦ * d2r, longθ * d2r}
}

// Unshamefully copied from smd/math.go
func cross(a, b []float64) []float64 {
	return []float64{a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0]} // Cross product R x V.
}
