package smd

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// TransferType defines the type of Lambert transfer
type TransferType uint8

// Longway returns whether or not this is the long way.
func (t TransferType) Longway() bool {
	switch t {
	case TType1:
		fallthrough
	case TType3:
		return false
	case TType2:
		fallthrough
	case TType4:
		return true
	default:
		panic(fmt.Errorf("cannot determine whether long or short way for %s", t))
	}
}

// Revs returns the number of revolutions given the type.
func (t TransferType) Revs() float64 {
	switch t {
	case TTypeAuto:
		fallthrough // auto-revs is limited to zero revolutions
	case TType1:
		fallthrough
	case TType2:
		return 0
	case TType3:
		fallthrough
	case TType4:
		return 1
	default:
		panic("unknown transfer type")
	}
}

func (t TransferType) String() string {
	switch t {
	case TTypeAuto:
		return "auto-revs"
	case TType1:
		return "type-1"
	case TType2:
		return "type-2"
	case TType3:
		return "type-3"
	case TType4:
		return "type-4"
	default:
		panic("unknown transfer type")
	}
}

const (
	// TTypeAuto lets the Lambert solver determine the type
	TTypeAuto TransferType = iota + 1
	// TType1 is transfer of type 1 (zero revolution, short way)
	TType1
	// TType2 is transfer of type 2 (zero revolution, long way)
	TType2
	// TType3 is transfer of type 3 (one revolutions, short way)
	TType3
	// TType4 is transfer of type 4 (one revolutions, long way)
	TType4
	lambertε         = 1e-4                   // General epsilon
	lambertTlambertε = 1e-4                   // Time epsilon
	lambertνlambertε = (5e-5 / 180) * math.Pi // 0.00005 degrees
)

// Hohmann computes an Hohmann transfer. It returns the departure and arrival velocities, and the time of flight.
// To get final computations:
// ΔvInit = vDepature - vI
// ΔvFinal = vArrival - vF
func Hohmann(rI, vI, rF, vF float64, body CelestialObject) (vDeparture, vArrival float64, tof time.Duration) {
	aTransfer := 0.5 * (rI + rF)
	vDeparture = math.Sqrt((2 * body.GM() / rI) - (body.GM() / aTransfer))
	vArrival = math.Sqrt((2 * body.GM() / rF) - (body.GM() / aTransfer))
	tof = time.Duration(math.Pi*math.Sqrt(math.Pow(aTransfer, 3)/body.GM())) * time.Second
	return
}

// Lambert solves the Lambert boundary problem:
// Given the initial and final radii and a central body, it returns the needed initial and final velocities
// along with φ which is the square of the difference in eccentric anomaly. Note that the direction of motion
// is computed directly in this function to simplify the generation of Pork chop plots.
func Lambert(Ri, Rf *mat64.Vector, Δt0 time.Duration, ttype TransferType, body CelestialObject) (Vi, Vf *mat64.Vector, φ float64, err error) {
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
	Δt0Sec := Δt0.Seconds()
	rI := mat64.Norm(Ri, 2)
	rF := mat64.Norm(Rf, 2)
	cosΔν := mat64.Dot(Ri, Rf) / (rI * rF)
	// Compute the direction of motion
	νI := math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
	νF := math.Atan2(Rf.At(1, 0), Rf.At(0, 0))
	dm := 1.0
	if ttype == TType2 {
		dm = -1.0
	} else if ttype == TTypeAuto {
		Δν := math.Atan2(Rf.At(1, 0), Rf.At(0, 0)) - math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
		if Δν > 2*math.Pi {
			Δν -= 2 * math.Pi
		} else if Δν < 0 {
			Δν += 2 * math.Pi
		}
		if Δν > math.Pi {
			dm = -1.0
		} // We don't do the < math.Pi case because that's the initial value anyway.
	}

	A := dm * math.Sqrt(rI*rF*(1+cosΔν))
	if νF-νI < lambertνlambertε && floats.EqualWithinAbs(A, 0, lambertε) {
		err = errors.New("cannot compute trajectory: Δν ~=0 and A ~=0")
		return
	}

	φup := 4 * math.Pow(math.Pi, 2) * math.Pow(ttype.Revs()+1, 2)
	φlow := -4 * math.Pi

	if ttype.Revs() > 0 {
		// Generate a bunch of φ
		Δtmin := 4000 * 24 * 3600.0
		φBound := 0.0

		for φP := 15.; φP < φup; φP += 0.1 {
			c2 := (1 - math.Cos(math.Sqrt(φP))) / φP
			c3 := (math.Sqrt(φP) - math.Sin(math.Sqrt(φP))) / math.Sqrt(math.Pow(φP, 3))
			y := rI + rF + A*(φP*c3-1)/math.Sqrt(c2)
			χ := math.Sqrt(y / c2)
			Δt := (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.μ)
			if Δtmin > Δt {
				Δtmin = Δt
				φBound = φP
			}
		}

		// Determine whether we are going up or down bounds.
		if ttype == TType3 {
			φlow = φup
			φup = φBound
		} else if ttype == TType4 {
			φlow = φBound
		}
	}
	// Initial guesses for c2 and c3
	c2 := 1 / 2.
	c3 := 1 / 6.
	var Δt, y float64
	var iteration uint
	for math.Abs(Δt-Δt0Sec) > lambertTlambertε {
		if iteration > 10000 {
			err = errors.New("did not converge after 10000 iterations")
			return
		}
		iteration++
		y = rI + rF + A*(φ*c3-1)/math.Sqrt(c2)
		if A > 0 && y < 0 {
			tmpIt := 0
			for y < 0 {
				φ += 0.1
				y = rI + rF + A*(φ*c3-1)/math.Sqrt(c2)
				if tmpIt > 10000 {
					err = errors.New("did not converge after 10000 attempts to increase φ")
					return
				}
				tmpIt++
			}
		}
		χ := math.Sqrt(y / c2)
		Δt = (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.μ)
		if ttype != TType3 {
			if Δt <= Δt0Sec {
				φlow = φ
			} else {
				φup = φ
			}
		} else {
			if Δt >= Δt0Sec {
				φlow = φ
			} else {
				φup = φ
			}
		}
		φ = (φup + φlow) / 2
		if φ > lambertε {
			sφ := math.Sqrt(φ)
			ssφ, csφ := math.Sincos(sφ)
			c2 = (1 - csφ) / φ
			c3 = (sφ - ssφ) / math.Sqrt(math.Pow(φ, 3))
		} else if φ < -lambertε {
			sφ := math.Sqrt(-φ)
			c2 = (1 - math.Cosh(sφ)) / φ
			c3 = (math.Sinh(sφ) - sφ) / math.Sqrt(math.Pow(-φ, 3))
		} else {
			c2 = 1 / 2.
			c3 = 1 / 6.
		}
	}
	f := 1 - y/rI
	gDot := 1 - y/rF
	g := (A * math.Sqrt(y/body.μ))
	// Compute velocities
	Rf2 := mat64.NewVector(3, nil)
	Vi.AddScaledVec(Rf, -f, Ri)
	Vi.ScaleVec(1/g, Vi)
	Rf2.ScaleVec(gDot, Rf)
	Vf.AddScaledVec(Rf2, -1, Ri)
	Vf.ScaleVec(1/g, Vf)
	return
}
