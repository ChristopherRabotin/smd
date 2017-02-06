package tools

import (
	"errors"
	"math"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// TODO: Convert times to time.Duration

const (
	ε  = 1e-6                   // General epsilon
	tε = 1e-6                   // Time epsilon (1e-6 seconds)
	νε = (5e-5 / 180) * math.Pi // 0.00005 degrees
)

// Lambert solves the Lambert boundary problem:
// Given the initial and final radii and a central body, it returns the needed initial and final velocities
// along with ψ which is the square of the difference in eccentric anomaly. Note that the direction of motion
// is computed directly in this function to simplify the generation of Pork chop plots.
func Lambert(Ri, Rf *mat64.Vector, Δt0, dm float64, body smd.CelestialObject) (Vi, Vf *mat64.Vector, ψ float64, err error) {
	// Initialize return variables
	Vi = mat64.NewVector(3, nil)
	Vf = mat64.NewVector(3, nil)
	// Sanity checks
	Rir, _ := Ri.Dims()
	Rfr, _ := Rf.Dims()
	if Rir != Rfr || Rir != 3 {
		err = errors.New("initial and final radii must be 3x1 vectors")
		return
	}
	rI := norm(Ri)
	rF := norm(Rf)
	cosΔν := mat64.Dot(Ri, Rf) / (rI * rF)
	// Compute the direction of motion
	νI := math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
	νF := math.Atan2(Rf.At(1, 0), Rf.At(0, 0))
	if dm == 0 {
		if νF-νI < math.Pi {
			dm = 1
		} else {
			dm = -1
		}
	} else if dm != 1 && dm != -1 {
		err = errors.New("direction of motion must be either 0, -1 or 1 (multi rev not supported)")
		return
	}
	A := dm * math.Sqrt(rI*rF*(1+cosΔν))
	if νF-νI < νε && floats.EqualWithinAbs(A, 0, ε) {
		err = errors.New("Δν ~=0 and A ~=0, cannot compute trajectory")
		return
	}
	ψ = 0
	ψup := 4 * math.Pow(math.Pi, 2)
	ψlow := -4 * math.Pi
	// Initial guesses for c2 and c3
	c2 := 1 / 2.
	c3 := 1 / 6.
	var Δt, y float64
	for math.Abs(Δt-Δt0) > tε {
		y = rI + rF + A*(ψ*c3-1)/math.Sqrt(c2)
		if A > 0 && y < 0 {
			panic("not yet implemented")
			// Well this is confusing... ψlow needs to be readjusted but it never is used.
			//ψlow = (0.8 / c3) * (1 - (math.Sqrt(c2)/A)*(rI+rF))
			//continue
		}
		χ := math.Sqrt(y / c2)
		Δt = (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.GM())
		if Δt < Δt0 {
			ψlow = ψ
		} else {
			ψup = ψ
		}
		ψ = (ψup + ψlow) / 2
		if ψ > ε {
			sψ := math.Sqrt(ψ)
			ssψ, csψ := math.Sincos(sψ)
			c2 = (1 - csψ) / ψ
			c3 = (sψ - ssψ) / math.Sqrt(math.Pow(ψ, 3))
		} else if ψ < -ε {
			sψ := math.Sqrt(-ψ)
			c2 = (1 - math.Cosh(sψ)) / ψ
			c3 = (math.Sinh(sψ) - sψ) / math.Sqrt(math.Pow(sψ, 3))
		} else {
			c2 = 1 / 2.
			c3 = 1 / 6.
		}
	}
	f := 1 - y/rI
	gDot := 1 - y/rF
	g := (A * math.Sqrt(y/body.GM()))
	// Compute velocities
	Rf2 := mat64.NewVector(3, nil)
	Vi.AddScaledVec(Rf, -f, Ri)
	Vi.ScaleVec(1/g, Vi)
	Rf2.ScaleVec(gDot, Rf)
	Vf.AddScaledVec(Rf2, -1, Ri)
	Vf.ScaleVec(1/g, Vf)
	return
}
