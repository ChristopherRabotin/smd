package smd

import "math"

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
