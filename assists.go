package smd

import (
	"errors"
	"math"
)

// BPlane stores B-plane parameters and allows for differential correction.
type BPlane struct {
	initOrbit                Orbit
	BR, BT, LTOF             float64
	goalBT, goalBR, goalLTOF float64
	tolBT, tolBR, tolLTOF    float64
}

// SetBTGoal sets to the B_T goal
func (b BPlane) SetBTGoal(value, tolerance float64) {}

// SetBRGoal sets to the B_R goal
func (b BPlane) SetBRGoal(value, tolerance float64) {}

// SetLTOFGoal sets to the LTOF goal
func (b BPlane) SetLTOFGoal(value, tolerance float64) {}

// AchieveGoals attempts to achieve the provided goals.
// Returns an error if no goal is set or is no convergence after a certain number
// of attempts. Otherwise, returns the Delta-V to apply.
func (b BPlane) AchieveGoals() ([]float64, error) {
	if math.IsNaN(b.goalBR) && math.IsNaN(b.goalBT) && math.IsNaN(b.goalLTOF) {
		return nil, errors.New("no goal set")
	}
	return nil, nil
}

// NewBPlane returns the B-plane of a given orbit.
func NewBPlane(o Orbit) BPlane {
	// Some of this is quite similar to RV2COE.
	hHat := unit(cross(o.rVec, o.vVec))
	k := []float64{0, 0, 1}
	v := norm(o.vVec)
	r := norm(o.rVec)
	eVec := make([]float64, 3, 3)
	for i := 0; i < 3; i++ {
		eVec[i] = ((v*v-o.Origin.μ/r)*o.rVec[i] - dot(o.rVec, o.vVec)*o.vVec[i]) / o.Origin.μ
	}
	e := norm(eVec)
	ξ := (v*v)/2 - o.Origin.μ/r
	a := -o.Origin.μ / (2 * ξ)
	c := a * e
	b := math.Sqrt(math.Pow(c, 2) - math.Pow(a, 2))

	// Compute B plane frame
	heVec := unit(cross(hHat, eVec))
	β := math.Acos(1 / e)
	sinβ, cosβ := math.Sincos(β)
	sHat := make([]float64, 3)
	for i := 0; i < 3; i++ {
		sHat[i] = cosβ*eVec[i]/e + sinβ*heVec[i]
	}
	tHat := unit(cross(sHat, k))
	rHat := unit(cross(sHat, tHat))
	bVec := cross(sHat, hHat)
	for i := 0; i < 3; i++ {
		bVec[i] *= b
	}
	bT := dot(bVec, tHat)
	bR := dot(bVec, rHat)
	νB := math.Pi/2 - β
	sinνB, cosνB := math.Sincos(νB)
	νR := math.Acos((-a*(e*e-1))/(r*e) - 1/e)
	sinνR, cosνR := math.Sincos(νR)

	fB := math.Asinh(sinνB*math.Sqrt(e*e-1)) / (1 + e*cosνB)
	fR := math.Asinh(sinνR*math.Sqrt(e*e-1)) / (1 + e*cosνR)
	ltof := ((e*math.Sinh(fB) - fB) - (e*math.Sinh(fR) - fR)) / o.MeanAnomaly()
	return BPlane{initOrbit: o, BR: bR, BT: bT, LTOF: ltof, goalBT: math.NaN(), goalBR: math.NaN(), goalLTOF: math.NaN()}
}

// GATurnAngle computes the turn angle about a given body based on the radius of periapsis.
func GATurnAngle(vInf, rP float64, body CelestialObject) float64 {
	ρ := math.Acos(1 / (1 + math.Pow(vInf, 2)*(rP/body.μ)))
	return math.Pi - 2*ρ
}

// GAFromVinf computes gravity assist parameters about a given body from the V infinity vectors.
// All angles are in radians!
func GAFromVinf(vInfInVec, vInfOutVec []float64, body CelestialObject) (ψ, rP, bT, bR, B, θ float64) {
	vInfIn := norm(vInfInVec)
	vInfOut := norm(vInfOutVec)
	ψ = math.Acos(dot(vInfInVec, vInfOutVec) / (vInfIn * vInfOut))
	rP = (body.μ / math.Pow(vInfIn, 2)) * (1/math.Cos((math.Pi-ψ)/2) - 1)
	k := []float64{0, 0, 1}
	sHat := unit(vInfInVec)
	tHat := unit(cross(sHat, k))
	rHat := unit(cross(sHat, tHat))
	hHat := unit(cross(vInfInVec, vInfOutVec))
	bVec := unit(cross(sHat, hHat))
	bVal := (body.μ / math.Pow(vInfIn, 2)) * math.Sqrt(math.Pow(1+math.Pow(vInfIn, 2)*(rP/body.μ), 2)-1)
	for i := 0; i < 3; i++ {
		bVec[i] *= bVal
	}
	bT = dot(bVec, tHat)
	bR = dot(bVec, rHat)
	B = norm(bVec)
	θ = math.Atan2(bT, bR)
	return
}
