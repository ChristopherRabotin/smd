package dynamics

import (
	"fmt"
	"math"

	"github.com/gonum/floats"
)

// ControlLaw defines an enum of control laws.
type ControlLaw uint8

// ControlLawType defines the way to sum different Lyuapunov optimal CL
type ControlLawType uint8

const (
	tangential ControlLaw = iota + 1
	antiTangential
	inversion
	coast
	multiOpti
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
	// Petropoulos idem as Ruggerio, but with Petropoulos
	Petropoulos
	// Naasz is another type of combination of control law
	Naasz
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
	}
	panic("cannot stringify unknown control law")
}

func (meth ControlLawType) String() string {
	switch meth {
	case Ruggerio:
		return "Ruggerio"
	case Naasz:
		return "Naasz"
	case Petropoulos:
		return "Petro"
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

// Thruster defines a thruster interface.
type Thruster interface {
	// Returns the minimum power and voltage requirements for this thruster.
	Min() (voltage, power uint)
	// Returns the max power and voltage requirements for this thruster.
	Max() (voltage, power uint)
	// Returns the thrust in Newtons and isp consumed in seconds.
	Thrust(voltage, power uint) (thrust, isp float64)
}

/* Available thrusters */

// PPS1350 is the Snecma thruster used on SMART-1.
// Source: http://www.esa.int/esapub/bulletin/bulletin129/bul129e_estublier.pdf
type PPS1350 struct{}

// Min implements the Thruster interface.
func (t *PPS1350) Min() (voltage, power uint) {
	return t.Max()
}

// Max implements the Thruster interface.
func (t *PPS1350) Max() (voltage, power uint) {
	return 350, 2500
}

// Thrust implements the Thruster interface.
func (t *PPS1350) Thrust(voltage, power uint) (thrust, isp float64) {
	if voltage == 350 && power == 2500 {
		//return 140 * 1e-3, 1800
		return 89e-3, 1650
	}
	panic("unsupported voltage or power provided")
}

// HPHET12k5 is based on the NASA & Rocketdyne 12.5kW demo
/*type HPHET12k5 struct{}

// Min implements the Thruster interface.
func (t *HPHET12k5) Min() (voltage, power uint) {
	return 400, 12500
}

// Max implements the Thruster interface.
func (t *HPHET12k5) Max() (voltage, power uint) {
	return 400, 12500
}

// Thrust implements the Thruster interface.
func (t *HPHET12k5) Thrust(voltage, power uint) (thrust, fuelMass float64) {
	if voltage == 400 && power == 12500 {
		return 0.680, 4.8 * 1e-5 // fuel usage made up assuming linear from power.
	}
	panic("unsupported voltage or power provided")
}*/

// GenericEP is a generic EP thruster.
type GenericEP struct {
	thrust float64
	isp    float64
}

// Min implements the Thruster interface.
func (t *GenericEP) Min() (voltage, power uint) {
	return 0, 0
}

// Max implements the Thruster interface.
func (t *GenericEP) Max() (voltage, power uint) {
	return 0, 0
}

// Thrust implements the Thruster interface.
func (t *GenericEP) Thrust(voltage, power uint) (thrust, isp float64) {
	return t.thrust, t.isp
}

// NewGenericEP returns a generic electric prop thruster.
func NewGenericEP(thrust, isp float64) *GenericEP {
	return &GenericEP{thrust, isp}
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
	return NewOptimalThrust(OptiΔaCL, cl.reason).Control(o)
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
	unitV := NewOptimalThrust(OptiΔaCL, cl.reason).Control(o)
	unitV[0] *= -1
	unitV[1] *= -1
	unitV[2] *= -1
	return unitV
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
	return Inversion{ν, GenericCL{inversion.String(), inversion}}
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
			_, cosE := o.GetSinCosE()
			sinν, cosν := math.Sincos(o.ν)
			// WARNING: Using Atan2 for quadrant check actually breaks things...
			return unitΔvFromAngles(math.Atan(sinν/(cosν+cosE)), 0.0)
		}
		break
	case OptiΔiCL:
		ctrl = func(o Orbit) []float64 {
			return unitΔvFromAngles(0.0, -math.Pi/2)
			//return unitΔvFromAngles(0.0, sign(math.Cos(o.ω+o.ν))*math.Pi/2)
		}
		break
	case OptiΔΩCL:
		ctrl = func(o Orbit) []float64 {
			return unitΔvFromAngles(0.0, sign(math.Sin(o.ω+o.ν))*math.Pi/2)
		}
		break
	case OptiΔωCL:
		// The argument of periapsis control is from Ruggerio. The one in Petropoulos
		// also changes other orbital elements, although it's much simpler to calculate.
		ctrl = func(o Orbit) []float64 {
			cotν := 1 / math.Tan(o.ν)
			coti := 1 / math.Tan(o.i)
			sinν, cosν := math.Sincos(o.ν)
			sinων := math.Sin(o.ω + o.ν)
			α := math.Atan2((1+o.e*cosν)*cotν, 2+o.e*cosν)
			sinαν := math.Sin(α - o.ν)
			β := math.Atan2(o.e*coti*sinων, sinαν*(1+o.e*cosν)-math.Cos(α)*sinν)
			return unitΔvFromAngles(α, β)
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
	ainit, atarget float64
	iinit, itarget float64
	einit, etarget float64
	Ωinit, Ωtarget float64
	ωinit, ωtarget float64
	controls       []ThrustControl
	method         ControlLawType
	GenericCL
}

// NewOptimalΔOrbit generates a new OptimalΔOrbit based on the provided target orbit.
func NewOptimalΔOrbit(target Orbit, method ControlLawType, laws []ControlLaw) *OptimalΔOrbit {
	cl := OptimalΔOrbit{}
	cl.cleared = false
	cl.method = method
	cl.atarget = target.a
	cl.etarget = target.e
	cl.itarget = target.i
	cl.ωtarget = target.ω
	cl.Ωtarget = target.Ω
	if len(laws) == 0 {
		laws = []ControlLaw{OptiΔaCL, OptiΔeCL, OptiΔiCL, OptiΔΩCL, OptiΔωCL}
	}
	cl.controls = make([]ThrustControl, len(laws))
	for i, law := range laws {
		cl.controls[i] = NewOptimalThrust(law, "multi-opti")
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
		cl.ainit = o.a
		cl.einit = o.e
		cl.iinit = o.i
		cl.Ωinit = o.Ω
		cl.ωinit = o.ω
		cl.Initd = true
		if len(cl.controls) == 5 {
			// Let's populate this with the appropriate laws, so we're resetting it.
			cl.controls = make([]ThrustControl, 0)
			if !floats.EqualWithinAbs(cl.ainit, cl.atarget, distanceε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔaCL, "Δa"))
			}
			if !floats.EqualWithinAbs(cl.einit, cl.etarget, eccentricityε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔeCL, "Δe"))
			}
			if !floats.EqualWithinAbs(cl.iinit, cl.itarget, angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔiCL, "Δi"))
			}
			if !floats.EqualWithinAbs(cl.Ωinit, cl.Ωtarget, angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔΩCL, "ΔΩ"))
			}
			if !floats.EqualWithinAbs(cl.ωinit, cl.ωtarget, angleε) {
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
			return (target - oscul) / (target - init)
		}

		for _, ctrl := range cl.controls {
			var oscul, init, target, tol float64
			switch ctrl.Type() {
			case OptiΔaCL:
				oscul = o.a
				init = cl.ainit
				target = cl.atarget
				tol = distanceε
			case OptiΔeCL:
				oscul = o.e
				init = cl.einit
				target = cl.etarget
				tol = eccentricityε
			case OptiΔiCL:
				oscul = o.i
				init = cl.iinit
				target = cl.itarget
				tol = angleε
			case OptiΔΩCL:
				oscul = o.Ω
				init = cl.Ωinit
				target = cl.Ωtarget
				tol = angleε
			case OptiΔωCL:
				oscul = o.ω
				init = cl.ωinit
				target = cl.ωtarget
				tol = angleε
			}
			fact := factor(oscul, init, target, tol)
			if fact != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				// JIT changes for Ruggerio out of plane thrust direction
				if target > oscul {
					if ctrl.Type() == OptiΔiCL || ctrl.Type() == OptiΔΩCL {
						tmpThrust[2] *= -1
					}
				} else {
					if ctrl.Type() == OptiΔaCL {
						tmpThrust[0] *= -1
						tmpThrust[1] *= -1
					}
				}
				for i := 0; i < 3; i++ {
					thrust[i] += fact * tmpThrust[i]
				}
			}
		}
	case Naasz:
		// Note that, as described in Hatten MSc. thesis, the summing method only
		// works one way (because of the δO^2) per OE. So I added the sign function
		// before that to fix it.
		for _, ctrl := range cl.controls {
			var weight, δO float64
			p := o.GetSemiParameter()
			h := o.GetH()
			sinω, cosω := math.Sincos(o.ω)
			switch ctrl.Type() {
			case OptiΔaCL:
				weight = math.Pow(h, 2) / (4 * math.Pow(o.a, 4) * math.Pow(1+o.e, 2))
				δO = o.a - cl.atarget
				if math.Abs(δO) < distanceε {
					δO = 0
				}
			case OptiΔeCL:
				weight = math.Pow(h, 2) / (4 * math.Pow(p, 2))
				δO = o.e - cl.etarget
				if math.Abs(δO) < eccentricityε {
					δO = 0
				}
			case OptiΔiCL:
				weight = math.Pow((h+o.e*h*math.Cos(o.ω+math.Asin(o.e*sinω)))/(p*(math.Pow(o.e*sinω, 2)-1)), 2)
				δO = o.i - cl.itarget
				if math.Abs(δO) < angleε {
					δO = 0
				}
			case OptiΔΩCL:
				weight = math.Pow((h*math.Sin(o.i)*(o.e*math.Sin(o.ω+math.Asin(o.e*cosω))-1))/(p*(1-math.Pow(o.e*cosω, 2))), 2)
				δO = o.Ω - cl.Ωtarget
				if math.Abs(δO) < angleε {
					δO = 0
				}
			case OptiΔωCL:
				weight = (math.Pow(o.e*h, 2) / (4 * math.Pow(p, 2))) * (1 - math.Pow(o.e, 2)/4)
				δO = o.ω - cl.ωtarget
				if math.Abs(δO) < angleε {
					δO = 0
				}
			}
			if δO != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				for i := 0; i < 3; i++ {
					thrust[i] += 0.5 * weight * math.Pow(δO, 2) * tmpThrust[i]
				}

			}
		}
	default:
		panic(fmt.Errorf("control law sumation %+v not yet supported", cl.method))
	}

	return unit(thrust)
}
