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
	lambertε         = 1e-6                   // General epsilon
	lambertTlambertε = 1e-6                   // Time epsilon (1e-6 seconds)
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
// along with ψ which is the square of the difference in eccentric anomaly. Note that the direction of motion
// is computed directly in this function to simplify the generation of Pork chop plots.
func Lambert(Ri, Rf *mat64.Vector, Δt0 time.Duration, ttype TransferType, body CelestialObject) (Vi, Vf *mat64.Vector, ψ float64, err error) {
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
	rI := mat64.Norm(Ri, 2)
	rF := mat64.Norm(Rf, 2)
	cosΔν := mat64.Dot(Ri, Rf) / (rI * rF)
	// Compute the direction of motion
	νI := math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
	νF := math.Atan2(Rf.At(1, 0), Rf.At(0, 0))
	dm := 1.0
	/*var dm float64
	switch ttype {
	case TType1:
		dm = 1
	case TType2:
		dm = -1
	default:
		if νF-νI < math.Pi {
			dm = 1
		} else {
			dm = -1
		}
	}*/
	A := dm * math.Sqrt(rI*rF*(1+cosΔν))
	/*if true {
		panic(fmt.Errorf("A=%f\trI=%f\trF=%f\tcos=%f\n", A, rI, rF, cosΔν))
	}*/
	if νF-νI < lambertνlambertε && floats.EqualWithinAbs(A, 0, lambertε) {
		err = errors.New("Δν ~=0 and A ~=0, cannot compute trajectory")
		return
	}
	ψ = 0
	ψup := 4 * math.Pow(math.Pi, 2) * math.Pow(ttype.Revs()+1, 2)
	ψlow := -4 * math.Pi
	if ttype.Revs() > 0 {
		ψlow = 4 * math.Pow(math.Pi, 2) * math.Pow(ttype.Revs(), 2)
	}
	// Initial guesses for c2 and c3
	c2 := 1 / 2.
	c3 := 1 / 6.
	var Δt, y float64
	Δt0Sec := Δt0.Seconds()
	var iteration uint
	for math.Abs(Δt-Δt0Sec) > lambertTlambertε {
		if iteration > 1000 {
			err = errors.New("did not converge after 1000 iterations")
			return
		}
		iteration++
		y = rI + rF + A*(ψ*c3-1)/math.Sqrt(c2)
		if A > 0 && y < 0 {
			//fmt.Printf("[%03d] y=%f\tA=%f\n", iteration, y, A)
			tmpIt := 0
			for y < 0 {
				ψ += 0.1
				y = rI + rF + A*(ψ*c3-1)/math.Sqrt(c2)
				if tmpIt > 1000 {
					err = errors.New("did not converge after 1000 attempts to increase ψ")
					return
				}
				tmpIt++
			}
		}
		χ := math.Sqrt(y / c2)
		//fmt.Printf("[%03d] y=%f\tχ=%f\n", iteration, y, χ)
		Δt = (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.μ)
		if Δt < Δt0Sec {
			ψlow = ψ
		} else {
			ψup = ψ
		}
		ψ = (ψup + ψlow) / 2
		if ψ > lambertε {
			sψ := math.Sqrt(ψ)
			ssψ, csψ := math.Sincos(sψ)
			c2 = (1 - csψ) / ψ // BUG: c2 may go to 0
			c3 = (sψ - ssψ) / math.Sqrt(math.Pow(ψ, 3))
			//fmt.Printf("[%03d] POS c2=%f\tc3=%f\tψ=%f\n", iteration, c2, c3, ψ)
		} else if ψ < -lambertε {
			sψ := math.Sqrt(-ψ)
			c2 = (1 - math.Cosh(sψ)) / ψ
			c3 = (math.Sinh(sψ) - sψ) / math.Sqrt(math.Pow(-ψ, 3))
			//fmt.Printf("[%03d] NEG c2=%f\tc3=%f\tψ=%f\n", iteration, c2, c3, ψ)
		} else {
			c2 = 1 / 2.
			c3 = 1 / 6.
			//fmt.Printf("[%03d] ZER c2=%f\tc3=%f\tψ=%f\n", iteration, c2, c3, ψ)
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

func x2tof(x, s, c float64, ttype TransferType) float64 {
	var am, a, alfa, beta float64

	am = s / 2
	a = am / (1 - x*x)
	if x < 1 {
		// Ellipse
		beta = 2 * math.Asin(math.Sqrt((s-c)/(2*a)))
		if ttype.Longway() {
			beta = -beta
		}
		alfa = 2 * math.Acos(x)
	} else {
		// Hyperbola
		alfa = 2 * math.Acosh(x)
		beta = 2 * math.Asinh(math.Sqrt((s-c)/(-2*a)))
		if ttype.Longway() {
			beta = -beta
		}
	}

	if a > 0 {
		return (a * math.Sqrt(a) * ((alfa - math.Sin(alfa)) - (beta - math.Sin(beta))))
	}
	return (-a * math.Sqrt(-a) * ((math.Sinh(alfa) - alfa) - (math.Sinh(beta) - beta)))

}

// LambertMultiRev is a multi revolution Lambert boundary solver. It a port from ESA's Advanced Concept Team and their C++ code AstroToolbox/Lambert.cpp.
func LambertMultiRev(Ri, Rf *mat64.Vector, Δt0 time.Duration, ttype TransferType, body CelestialObject) (Vi, Vf *mat64.Vector, ψ float64, err error) {
	if Δt0 < 0 {
		panic("Δt must be positive")
	}
	lw := ttype.Longway()

	//r1Norm := mat64.Norm(Ri, 2)
	r2Norm := mat64.Norm(Rf, 2)
	//V := body.μ / r1Norm
	//T := r1Norm / V

	// working with non-dimensional radii and time-of-flight
	t := Δt0.Seconds()
	//Ri.ScaleVec(1/r1Norm, Ri)
	//Rf.ScaleVec(1/r1Norm, Rf)

	theta := math.Acos(mat64.Dot(Ri, Rf) / r2Norm)

	if lw {
		theta = 2*math.Acos(-1.0) - theta
	}

	c := math.Sqrt(1 + r2Norm*(r2Norm-2.0*math.Cos(theta)))
	s := (1 + r2Norm + c) / 2.0
	am := s / 2.0
	lambda := math.Sqrt(r2Norm) * math.Cos(theta/2.0) / s

	// We start finding the log(x+1) value of the solution conic:
	// NO MULTI REV --> (1 SOL)
	//	inn1=-.5233;    //first guess point
	//  inn2=.5233;     //second guess point
	x1 := math.Log(0.4767)
	x2 := math.Log(1.5233)
	y1 := math.Log(x2tof(-.5233, s, c, ttype)) - math.Log(t)
	y2 := math.Log(x2tof(.5233, s, c, ttype)) - math.Log(t)

	// Regula-falsi iterations
	deltaErr := 1.0
	iter := 0
	var newX, newY float64
	tolerance := 1e-11
	for (deltaErr > tolerance) && (y1 != y2) {
		iter++
		newX = (x1*y2 - y1*x2) / (y2 - y1)
		newY = math.Log(x2tof(math.Exp(newX)-1, s, c, ttype)) - math.Log(t)
		x1 = x2
		y1 = y2
		x2 = newX
		y2 = newY
		deltaErr = math.Abs(x1 - newX)
	}
	fmt.Printf("converged in %d iterations\n", iter)
	x := math.Exp(newX) - 1

	// The solution has been evaluated in terms of log(x+1) or tan(x*pi/2), we
	// now need the conic. As for transfer angles near to pi the lagrange
	// coefficient technique goes singular (dg approaches a zero/zero that is
	// numerically bad) we here use a different technique for those cases. When
	// the transfer angle is exactly equal to pi, then the ih unit vector is not
	// determined. The remaining equations, though, are still valid.

	a := am / (1 - x*x)

	// psi evaluation
	var alfa, beta, eta, eta2, psi float64
	if x < 1 { // ellipse
		beta = 2 * math.Asin(math.Sqrt((s-c)/(2*a)))
		if ttype.Longway() {
			beta = -beta
		}
		alfa = 2 * math.Acos(x)
		psi = (alfa - beta) / 2
		eta2 = 2 * a * math.Pow(math.Sin(psi), 2) / s
		eta = math.Sqrt(eta2)
	} else { // hyperbola
		beta = 2 * math.Asinh(math.Sqrt((c-s)/(2*a)))
		if ttype.Longway() {
			beta = -beta
		}
		alfa = 2 * math.Acosh(x)
		psi = (alfa - beta) / 2
		eta2 = -2 * a * math.Pow(math.Sinh(psi), 2) / s
		eta = math.Sqrt(eta2)
	}

	// parameter of the solution
	p := (r2Norm / (am * eta2)) * math.Pow(math.Sin(theta/2), 2)
	sigma1 := (1 / (eta * math.Sqrt(am))) * (2*lambda*am - (lambda + x*eta))
	ih := unitVec(crossVec(Ri, Rf))

	if ttype.Longway() {
		ih.ScaleVec(-1, ih)
	}

	vr1 := sigma1
	vt1 := math.Sqrt(p)
	dum := crossVec(ih, Ri)

	v1 := mat64.NewVector(3, nil)
	for i := 0; i < 3; i++ {
		v1.SetVec(i, vr1*Ri.At(i, 0)+vt1*dum.At(i, 0))
	}

	vt2 := vt1 / r2Norm
	vr2 := -vr1 + (vt1-vt2)/math.Tan(theta/2)
	dum = crossVec(ih, unitVec(Rf))

	v2 := mat64.NewVector(3, nil)
	for i := 0; i < 3; i++ {
		v2.SetVec(i, vr2*Rf.At(i, 0)/r2Norm+vt2*dum.At(i, 0))
	}
	//v1.ScaleVec(V, v1)
	//v2.ScaleVec(V, v2)
	//a *= r1Norm
	//p *= r1Norm
	fmt.Printf("a=%f\tp=%f\tpsi=%f\n", a, p, psi)
	return v1, v2, psi, nil
}
