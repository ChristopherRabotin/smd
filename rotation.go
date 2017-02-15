package smd

import (
	"math"

	"github.com/gonum/matrix/mat64"
)

const (
	// EarthRotationRate is the average Earth rotation rate in radians per second.
	EarthRotationRate = 7.2921158553e-5
)

// Rot313Vec converts a given vector from PQW frame to ECI frame.
func Rot313Vec(θ1, θ2, θ3 float64, vI []float64) []float64 {
	return MxV33(R3R1R3(θ1, θ2, θ3), vI)
}

// R3R1R3 performs a 3-1-3 Euler parameter rotation.
// From Schaub and Junkins (the one in Vallado is wrong... surprinsingly, right? =/)
func R3R1R3(θ1, θ2, θ3 float64) *mat64.Dense {
	sθ1, cθ1 := math.Sincos(θ1)
	sθ2, cθ2 := math.Sincos(θ2)
	sθ3, cθ3 := math.Sincos(θ3)
	return mat64.NewDense(3, 3, []float64{cθ3*cθ1 - sθ3*cθ2*sθ1, cθ3*sθ1 + sθ3*cθ2*cθ1, sθ3 * sθ2,
		-sθ3*cθ1 - cθ3*cθ2*sθ1, -sθ3*sθ1 + cθ3*cθ2*cθ1, cθ3 * sθ2,
		sθ2 * sθ1, -sθ2 * cθ1, cθ2})
}

// R1 rotation about the 1st axis.
func R1(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{1, 0, 0, 0, c, s, 0, -s, c})
}

// R2 rotation about the 2nd axis.
func R2(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{c, 0, -s, 0, 1, 0, s, 0, c})
}

// R3 rotation about the 3rd axis.
func R3(x float64) *mat64.Dense {
	s, c := math.Sincos(x)
	return mat64.NewDense(3, 3, []float64{c, s, 0, -s, c, 0, 0, 0, 1})
}

// MxV33 multiplies a matrix with a vector. Note that there is no dimension check!
func MxV33(m *mat64.Dense, v []float64) (o []float64) {
	vVec := mat64.NewVector(len(v), v)
	var rVec mat64.Vector
	rVec.MulVec(m, vVec)
	return []float64{rVec.At(0, 0), rVec.At(1, 0), rVec.At(2, 0)}
}

// GEO2ECEF converts the provided parameters (in km and radians) to the ECEF vector.
// Note that the first parameter is the altitude, not the radius from the center of the body!
func GEO2ECEF(altitude, latitude, longitude float64) []float64 {
	sLong, cLong := math.Sincos(longitude)
	sLat, cLat := math.Sincos(latitude)
	r := altitude + Earth.Radius
	return []float64{r * cLat * cLong, r * cLat * sLong, r * sLat}
}

// ECI2ECEF converts the provided ECI vector to ECEF for the θgst given in degrees.
func ECI2ECEF(R []float64, θgst float64) []float64 {
	return MxV33(R3(θgst), R)
}

// ECEF2ECI converts the provided ECEF vector to ECI for the θgst given in degrees.
func ECEF2ECI(R []float64, θgst float64) []float64 {
	return ECI2ECEF(R, -θgst)
}
