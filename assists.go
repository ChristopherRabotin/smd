package smd

import (
	"errors"
	"fmt"
	"math"

	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// BPlane stores B-plane parameters and allows for differential correction.
type BPlane struct {
	Orbit                    Orbit
	BR, BT, LTOF             float64
	goalBT, goalBR, goalLTOF float64
	tolBT, tolBR, tolLTOF    float64
}

// attemptWithinGoal returns whether the provided B-plane is equal within the tolerance
// of the receiver.
func (b BPlane) attemptWithinGoal(attempt BPlane) bool {
	if !b.anyGoalSet() {
		return false
	}
	if !math.IsNaN(b.goalBR) && !floats.EqualWithinAbs(b.goalBR, attempt.BR, b.tolBR) {
		return false
	}
	if !math.IsNaN(b.goalBT) && !floats.EqualWithinAbs(b.goalBT, attempt.BT, b.tolBT) {
		return false
	}
	if !math.IsNaN(b.goalLTOF) && !floats.EqualWithinAbs(b.goalLTOF, attempt.LTOF, b.tolLTOF) {
		return false
	}
	return true
}

// SetBTGoal sets to the B_T goal
func (b *BPlane) SetBTGoal(value, tolerance float64) {
	b.goalBT = value
	b.tolBT = tolerance
}

// SetBRGoal sets to the B_R goal
func (b *BPlane) SetBRGoal(value, tolerance float64) {
	b.goalBR = value
	b.tolBR = tolerance
}

// SetLTOFGoal sets to the LTOF goal
func (b *BPlane) SetLTOFGoal(value, tolerance float64) {
	b.goalLTOF = value
	b.tolLTOF = tolerance
}

func (b BPlane) anyGoalSet() bool {
	return !(math.IsNaN(b.goalBR) && math.IsNaN(b.goalBT) && math.IsNaN(b.goalLTOF))
}

// AchieveGoals attempts to achieve the provided goals.
// Returns an error if no goal is set or is no convergence after a certain number
// of attempts. Otherwise, returns the velocity vector needed to reach the goal.
func (b BPlane) AchieveGoals(components int) ([]float64, error) {
	if components < 2 || components > 3 {
		panic("components must be 2 or 3")
	}
	if !b.anyGoalSet() {
		return nil, errors.New("no goal set")
	}
	var converged = false
	var R, V = b.Orbit.RV()
	pert := math.Pow(10, -10)
	for iter := 0; iter < 100; iter++ {
		// Compute updated B plane
		nominal := NewBPlane(*NewOrbitFromRV(R, V, Earth))
		// Update the nominal values
		converged = b.attemptWithinGoal(nominal)
		if converged {
			break
		}
		// Vary velocity vector and build the Jacobian
		jacob := mat64.NewDense(components, components, nil)
		for i := 0; i < components; i++ { // Vx, Vy, Vz
			vTmp := make([]float64, 3)
			copy(vTmp, V)
			vTmp[i] += pert
			attempt := NewBPlane(*NewOrbitFromRV(R, vTmp, Earth))
			// Compute Jacobian
			// BT, BR, LTOF
			jacob.Set(0, i, (attempt.BT-nominal.BT)/pert)
			jacob.Set(1, i, (attempt.BR-nominal.BR)/pert)
			if components > 2 {
				jacob.Set(i, 2, (nominal.LTOF-attempt.LTOF)/pert)
			}
		}
		// Invert Jacobian
		var invJacob mat64.Dense
		if err := invJacob.Inverse(jacob); err != nil {
			fmt.Printf("%+v\n", mat64.Formatted(jacob))
			panic("could not invert jacobian!")
		}
		ΔB := mat64.NewVector(components, nil)
		if !math.IsNaN(b.goalBT) {
			ΔB.SetVec(0, b.goalBT-nominal.BT)
		}
		if !math.IsNaN(b.goalBR) {
			ΔB.SetVec(1, b.goalBR-nominal.BR)
		}
		if components > 2 && !math.IsNaN(b.goalLTOF) {
			ΔB.SetVec(2, b.goalLTOF-nominal.LTOF)
		}
		var Δv mat64.Vector
		Δv.MulVec(&invJacob, ΔB)
		for i := 0; i < components; i++ {
			V[i] += Δv.At(i, 0)
		}
	}
	if !converged {
		return nil, errors.New("did not converge after 100 iterations")
	}
	return V, nil
}

func (b BPlane) String() string {
	return fmt.Sprintf("BR=%.8f\tBT=%.8f", b.BR, b.BT)
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
	return BPlane{Orbit: o, BR: bR, BT: bT, LTOF: ltof, goalBT: math.NaN(), goalBR: math.NaN(), goalLTOF: math.NaN()}
}

// GATurnAngle computes the turn angle about a given body based on the radius of periapsis.
func GATurnAngle(vInf, rP float64, body CelestialObject) float64 {
	ρ := math.Acos(1 / (1 + math.Pow(vInf, 2)*(rP/body.μ)))
	return math.Pi - 2*ρ
}

// GARPeriapsis computes the radius of periapsis from the turn angle about a given body.
func GARPeriapsis(vInf, ψ float64, body CelestialObject) float64 {
	cosρ := math.Cos(0.5 * (math.Pi - ψ))
	return (1/cosρ - 1) * body.μ / math.Pow(vInf, 2)
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
