package smd

import (
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

// ControlLaw defines an enum of control laws.
type ControlLaw uint8

// ControlLawType defines the way to sum different Lyuapunov optimal CL
type ControlLawType uint8

type hohmannStatus uint8

const (
	tangential ControlLaw = iota + 1
	antiTangential
	inversion
	coast
	multiOpti
	hohmann
	// OptiΔaCL allows to optimize thrust for semi major axis change
	OptiΔaCL
	// OptiΔiCL allows to optimize thrust for inclination change
	OptiΔiCL
	// OptiΔeCL allows to optimize thrust for eccentricity change
	OptiΔeCL
	// OptiΔΩCL allows to optimize thrust forRAAN change
	OptiΔΩCL
	// OptiΔωCL allows to optimize thrust for argument of perigee change
	OptiΔωCL
	// Ruggerio uses the eponym method of combining the control laws
	Ruggerio ControlLawType = iota + 1
	// Naasz is another type of combination of control law
	Naasz
	hohmannCompute hohmannStatus = iota + 1
	hohmmanInitΔv
	hohmmanFinalΔv
	hohmmanCoast
	hohmmanCompleted
)

func (cl ControlLaw) String() string {
	switch cl {
	case tangential:
		return "tan"
	case antiTangential:
		return "aTan"
	case inversion:
		return "inversion"
	case coast:
		return "coast"
	case OptiΔaCL:
		return "optiΔa"
	case OptiΔeCL:
		return "optiΔe"
	case OptiΔiCL:
		return "optiΔi"
	case OptiΔΩCL:
		return "optiΔΩ"
	case OptiΔωCL:
		return "optiΔω"
	case multiOpti:
		return "multiOpti"
	case hohmann:
		return "Hohmann"
	}
	panic("cannot stringify unknown control law")
}

func (meth ControlLawType) String() string {
	switch meth {
	case Ruggerio:
		return "Ruggerio"
	case Naasz:
		return "Naasz"
	}
	panic("cannot stringify unknown control law summation method")
}

// ThrustControl defines a thrust control interface.
type ThrustControl interface {
	Control(o Orbit) []float64
	Type() ControlLaw
	Reason() string
}

// GenericCL partially defines a ThrustControl.
type GenericCL struct {
	reason string
	cl     ControlLaw
}

// Reason implements the ThrustControl interface.
func (cl GenericCL) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl GenericCL) Type() ControlLaw {
	return cl.cl
}

func newGenericCLFromCL(cl ControlLaw) GenericCL {
	return GenericCL{cl.String(), cl}
}

/* Let's define some control laws. */

// Coast defines an thrust control law which does not thrust.
type Coast struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl Coast) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl Coast) Type() ControlLaw {
	return coast
}

// Control implements the ThrustControl interface.
func (cl Coast) Control(o Orbit) []float64 {
	return []float64{0, 0, 0}
}

// Tangential defines a tangential thrust control law
type Tangential struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl Tangential) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl Tangential) Type() ControlLaw {
	return tangential
}

// Control implements the ThrustControl interface.
func (cl Tangential) Control(o Orbit) []float64 {
	return []float64{0, 1, 0}
}

// AntiTangential defines an antitangential thrust control law
type AntiTangential struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl AntiTangential) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl AntiTangential) Type() ControlLaw {
	return antiTangential
}

// Control implements the ThrustControl interface.
func (cl AntiTangential) Control(o Orbit) []float64 {
	return []float64{0, -1, 0}
}

// Inversion keeps the thrust as tangential but inverts its direction within an angle from the orbit apogee.
// This leads to collisions with main body if the orbit isn't circular enough.
// cf. Izzo et al. (https://arxiv.org/pdf/1602.00849v2.pdf)
type Inversion struct {
	ν float64
	GenericCL
}

// Control implements the ThrustControl interface.
func (cl Inversion) Control(o Orbit) []float64 {
	f := o.ν
	if o.e > 0.01 || (f > cl.ν-math.Pi && f < math.Pi-cl.ν) {
		return Tangential{}.Control(o)
	}
	return AntiTangential{}.Control(o)
}

// NewInversionCL defines a new inversion control law.
func NewInversionCL(ν float64) Inversion {
	return Inversion{ν, newGenericCLFromCL(inversion)}
}

/* Following optimal thrust change are from IEPC 2011's paper:
Low-Thrust Maneuvers for the Efficient Correction of Orbital Elements
A. Ruggiero, S. Marcuccio and M. Andrenucci */

func unitΔvFromAngles(α, β float64) []float64 {
	sinα, cosα := math.Sincos(α)
	sinβ, cosβ := math.Sincos(β)
	return []float64{sinα * cosβ, cosα * cosβ, sinβ}
}

// OptimalThrust is an optimal thrust.
type OptimalThrust struct {
	ctrl func(o Orbit) []float64
	GenericCL
}

// Control implements the ThrustControl interface.
func (cl OptimalThrust) Control(o Orbit) []float64 {
	return cl.ctrl(o)
}

// NewOptimalThrust returns a new optimal Δe.
func NewOptimalThrust(cl ControlLaw, reason string) ThrustControl {
	var ctrl func(o Orbit) []float64
	switch cl {
	case OptiΔaCL:
		ctrl = func(o Orbit) []float64 {
			sinν, cosν := math.Sincos(o.ν)
			return unitΔvFromAngles(math.Atan2(o.e*sinν, 1+o.e*cosν), 0.0)
		}
		break
	case OptiΔeCL:
		ctrl = func(o Orbit) []float64 {
			_, cosE := o.SinCosE()
			sinν, cosν := math.Sincos(o.ν)
			return unitΔvFromAngles(math.Atan2(sinν, cosν+cosE), 0.0)
		}
		break
	case OptiΔiCL:
		ctrl = func(o Orbit) []float64 {
			return unitΔvFromAngles(0.0, sign(math.Cos(o.ω+o.ν))*math.Pi/2)
		}
		break
	case OptiΔΩCL:
		ctrl = func(o Orbit) []float64 {
			return unitΔvFromAngles(0.0, sign(math.Sin(o.ω+o.ν))*math.Pi/2)
		}
		break
	case OptiΔωCL:
		// The argument of periapsis control is from Petropoulos and in plane.
		// The out of plane will change other orbital elements at the same time.
		// We determine which one to use based on the efficiency of each.
		ctrl = func(o Orbit) []float64 {
			oe2 := 1 - math.Pow(o.e, 2)
			e3 := math.Pow(o.e, 3)
			νOptiα := math.Acos(math.Pow(oe2/(2*e3)+math.Sqrt(0.25*math.Pow(oe2/e3, 2)+1/27.), 1/3.) - math.Pow(-oe2/(2*e3)+math.Sqrt(0.25*math.Pow(oe2/e3, 2)+1/27.), 1/3.) - 1/o.e)
			νOptiβ := math.Acos(-o.e*math.Cos(o.ω)) - o.ω
			if math.Abs(o.ν-νOptiα) < math.Abs(o.ν-νOptiβ) {
				// The true anomaly is closer to the optimal in plane thrust, so let's do an in-plane thrust.
				p := o.SemiParameter()
				sinν, cosν := math.Sincos(o.ν)
				return unitΔvFromAngles(math.Atan2(-p*cosν, (p+o.RNorm())*sinν), 0.0)
			}
			return unitΔvFromAngles(0.0, sign(-math.Sin(o.ω+o.ν))*math.Cos(o.i)*math.Pi/2)
		}
		break
	default:
		panic(fmt.Errorf("optmized %s not yet implemented", cl))
	}
	return OptimalThrust{ctrl, GenericCL{reason, cl}}
}

// OptimalΔOrbit combines all the control laws from Ruggiero et al.
type OptimalΔOrbit struct {
	Initd, cleared bool
	oInit, oTgt    Orbit //local copy of the initial and target orbits.
	controls       []ThrustControl
	method         ControlLawType
	GenericCL
}

// NewOptimalΔOrbit generates a new OptimalΔOrbit based on the provided target orbit.
func NewOptimalΔOrbit(target Orbit, method ControlLawType, laws []ControlLaw) *OptimalΔOrbit {
	cl := OptimalΔOrbit{}
	cl.cleared = false
	cl.method = method
	cl.oTgt = target
	if len(laws) == 0 {
		laws = []ControlLaw{OptiΔaCL, OptiΔeCL, OptiΔiCL, OptiΔΩCL, OptiΔωCL}
	}
	cl.controls = make([]ThrustControl, len(laws))
	for i, law := range laws {
		cl.controls[i] = NewOptimalThrust(law, law.String())
	}
	if len(cl.controls) > 1 {
		cl.GenericCL = GenericCL{"ΔOrbit", multiOpti}
	} else {
		cl.GenericCL = GenericCL{"ΔOrbit", cl.controls[0].Type()}
	}
	return &cl
}

func (cl *OptimalΔOrbit) String() string {
	return "OptimalΔOrbit"
}

// Control implements the ThrustControl interface.
func (cl *OptimalΔOrbit) Control(o Orbit) []float64 {
	thrust := []float64{0, 0, 0}
	if !cl.Initd {
		cl.Initd = true
		cl.oInit = o
		if len(cl.controls) == 5 {
			// Let's populate this with the appropriate laws, so we're resetting it.
			cl.controls = make([]ThrustControl, 0)
			if !floats.EqualWithinAbs(cl.oInit.a, cl.oTgt.a, distanceε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔaCL, "Δa"))
			}
			if !floats.EqualWithinAbs(cl.oInit.e, cl.oTgt.e, eccentricityε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔeCL, "Δe"))
			}
			if !floats.EqualWithinAbs(cl.oInit.i, cl.oTgt.i, angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔiCL, "Δi"))
			}
			if !floats.EqualWithinAbs(cl.oInit.Ω, cl.oTgt.Ω, angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔΩCL, "ΔΩ"))
			}
			if !floats.EqualWithinAbs(cl.oInit.ω, cl.oTgt.ω, angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔωCL, "Δω"))
			}
		}
		return thrust
	}

	cl.cleared = true
	switch cl.method {
	case Ruggerio:
		factor := func(oscul, init, target, tol float64) float64 {
			if floats.EqualWithinAbs(init, target, tol) || floats.EqualWithinAbs(oscul, target, tol) {
				return 0 // Don't want no NaNs now.
			}
			return (target - oscul) / math.Abs(target-init)
		}

		for _, ctrl := range cl.controls {
			var oscul, init, target, tol float64
			switch ctrl.Type() {
			case OptiΔaCL:
				oscul = o.a
				init = cl.oInit.a
				target = cl.oTgt.a
				tol = distanceε
			case OptiΔeCL:
				oscul = o.e
				init = cl.oInit.e
				target = cl.oTgt.e
				tol = eccentricityε
			case OptiΔiCL:
				oscul = o.i
				init = cl.oInit.i
				target = cl.oTgt.i
				tol = angleε
			case OptiΔΩCL:
				oscul = o.Ω
				init = cl.oInit.Ω
				target = cl.oTgt.Ω
				tol = angleε
			case OptiΔωCL:
				oscul = o.ω
				init = cl.oInit.ω
				target = cl.oTgt.ω
				tol = angleε
			}
			// XXX: This summation may be wrong: |\sum x_i| != \sum |x_i|.
			if fact := factor(oscul, init, target, tol); fact != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				for i := 0; i < 3; i++ {
					thrust[i] += fact * tmpThrust[i]
				}
			}
		}
	case Naasz:
		// Note that, as described in Hatten MSc. thesis, the summing method only
		// works one way (because of the δO^2) per OE. So I added the sign function
		// *every here and there* as needed that to fix it.
		for _, ctrl := range cl.controls {
			var weight, δO float64
			p := o.SemiParameter()
			h := o.HNorm()
			sinω, cosω := math.Sincos(o.ω)
			switch ctrl.Type() {
			case OptiΔaCL:
				δO = o.a - cl.oTgt.a
				if math.Abs(δO) < distanceε {
					δO = 0
				}
				weight = sign(-δO) * math.Pow(h, 2) / (4 * math.Pow(o.a, 4) * math.Pow(1+o.e, 2))
			case OptiΔeCL:
				δO = o.e - cl.oTgt.e
				if math.Abs(δO) < eccentricityε {
					δO = 0
				}
				weight = sign(-δO) * math.Pow(h, 2) / (4 * math.Pow(p, 2))
			case OptiΔiCL:
				δO = o.i - cl.oTgt.i
				if math.Abs(δO) < angleε {
					δO = 0
				}
				weight = sign(-δO) * math.Pow((h+o.e*h*math.Cos(o.ω+math.Asin(o.e*sinω)))/(p*(math.Pow(o.e*sinω, 2)-1)), 2)
			case OptiΔΩCL:
				δO = o.Ω - cl.oTgt.Ω
				if δO > math.Pi {
					// Enforce short path to correct angle.
					δO *= -1
				}
				if math.Abs(δO) < angleε {
					δO = 0
				}
				weight = sign(-δO) * math.Pow((h*math.Sin(o.i)*(o.e*math.Sin(o.ω+math.Asin(o.e*cosω))-1))/(p*(1-math.Pow(o.e*cosω, 2))), 2)
			case OptiΔωCL:
				δO = o.ω - cl.oTgt.ω
				if δO > math.Pi {
					// Enforce short path to correct angle.
					δO *= -1
				}
				if math.Abs(δO) < angleε {
					δO = 0
				}
				weight = sign(-δO) * (math.Pow(o.e*h, 2) / (4 * math.Pow(p, 2))) * (1 - math.Pow(o.e, 2)/4)
			}
			if δO != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				fact := 0.5 * weight * math.Pow(δO, 2)
				for i := 0; i < 3; i++ {
					thrust[i] += fact * tmpThrust[i]
				}
			}
		}
	default:
		panic(fmt.Errorf("control law sumation %+v not yet supported", cl.method))
	}

	return unit(thrust)
}

// HohmannΔv computes the Δv needed to go from one orbit to another, and performs an instantaneous Δv.
type HohmannΔv struct {
	target                      Orbit
	status                      hohmannStatus
	ΔvBurnInit, ΔvInit, ΔvFinal float64
	tof                         time.Duration
	GenericCL
}

// Precompute computes and displays the Hohmann transfer orbit.
func (cl *HohmannΔv) Precompute(o Orbit) {
	if !floats.EqualWithinAbs(cl.target.ν, o.ν, angleε) && !floats.EqualWithinAbs(cl.target.ν, o.ν+math.Pi, angleε) && !floats.EqualWithinAbs(cl.target.ν, o.ν-math.Pi, angleε) {
		panic(fmt.Errorf("cannot perform Hohmann between orbits with misaligned semi-major axes\nini: %s\ntgt: %s\n", o, cl.target))
	}
	if !floats.EqualWithinAbs(o.e, 0, eccentricityε) {
		panic(fmt.Errorf("cannot perform Hohmann from a non elliptical orbit"))
	}
	if !floats.EqualWithinAbs(cl.target.i, o.i, angleε) {
		panic(fmt.Errorf("cannot perform Hohmann between non co-planar orbits\nini: %s\ntgt: %s\n", o, cl.target))
	}
	if !floats.EqualWithinAbs(o.ν, 0, angleε) && !floats.EqualWithinAbs(o.ν, math.Pi, angleε) {
		fmt.Printf("[WARNING] Hohmann transfer started neither at apoapsis nor at periapasis (inefficient)\n")
	}
	rInit := o.RNorm()
	rFinal := cl.target.RNorm()
	vInit := o.VNorm()
	vFinal := cl.target.VNorm()
	vDeparture, vArrival, tof := Hohmann(rInit, vInit, rFinal, vFinal, o.Origin)
	cl.ΔvInit = vDeparture - vInit
	cl.ΔvFinal = vArrival - vFinal
	cl.tof = tof
	durStr := cl.tof.String() + fmt.Sprintf(" (~%.1fd)", cl.tof.Hours()/24)
	fmt.Printf("=== HOHMANN TRANSFER INFO ===\nHohmann transfer information - T.O.F.: %s\nvInit=%f km/s\tvFinal=%f km/s\nvDeparture=%f km/s\t vArrival=%f km/s\nΔvInit=%f km/s\tΔvFinal=%f\n=== HOHMANN TRANSFER END ====\n", durStr, vInit, vFinal, vDeparture, vArrival, cl.ΔvInit, cl.ΔvFinal)
}

// Control implements the ThrustControl interface.
func (cl *HohmannΔv) Control(o Orbit) []float64 {

	switch cl.status {
	case hohmmanCoast:
		fallthrough
	case hohmmanCompleted:
		return []float64{0, 0, 0}
	case hohmmanInitΔv:
		if floats.EqualWithinAbs(cl.ΔvBurnInit-o.VNorm(), cl.ΔvInit, velocityε) {
			// We have applied enough Δv, so let's stop burning.
			cl.status = hohmmanCoast
			return []float64{0, 0, 0}
		}
		return []float64{sign(cl.ΔvInit), 0, 0}
	case hohmmanFinalΔv:
		if floats.EqualWithinAbs(cl.ΔvBurnInit-o.VNorm(), cl.ΔvFinal, velocityε) {
			// We have applied enough Δv, so let's stop burning.
			cl.status = hohmmanCompleted
			cl.ΔvBurnInit = 0 // Reset to zero after burn is completed.
			return []float64{0, 0, 0}
		}
		return []float64{sign(cl.ΔvFinal), 0, 0}
	default:
		panic("unreachable code")
	}
}

// NewHohmannΔv defines a new inversion control law.
func NewHohmannΔv(target Orbit) HohmannΔv {
	if !floats.EqualWithinAbs(target.e, 0, eccentricityε) {
		panic(fmt.Errorf("cannot perform Hohmann to a non elliptical orbit"))
	}
	return HohmannΔv{target, hohmannCompute, 0, 0, 0, time.Duration(-1) * time.Second, newGenericCLFromCL(hohmann)}
}
