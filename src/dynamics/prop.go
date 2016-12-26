package dynamics

import (
	"fmt"
	"math"

	"github.com/gonum/floats"
)

// ControlLaw defines an enum of control laws.
type ControlLaw uint8

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
	OptigΔeCL
	// OptiΔΩCL allows to optimize thrust forRAAN change
	OptiΔΩCL
	// OptiΔωCL allows to optimize thrust for argument of perigee change
	OptiΔωCL
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
	case OptigΔeCL:
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
func (t *PPS1350) Thrust(voltage, power uint) (thrust, fuelMass float64) {
	if voltage == 350 && power == 2500 {
		return 140 * 1e-3, 1800
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
	ν      float64
	reason string
}

// Reason implements the ThrustControl interface.
func (cl Inversion) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl Inversion) Type() ControlLaw {
	return inversion
}

// Control implements the ThrustControl interface.
func (cl Inversion) Control(o Orbit) []float64 {
	f := o.ν
	if o.e > 0.01 || (f > cl.ν-math.Pi && f < math.Pi-cl.ν) {
		return Tangential{}.Control(o)
	}
	return AntiTangential{}.Control(o)
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
	case OptigΔeCL:
		ctrl = func(o Orbit) []float64 {
			_, cosE := o.GetSinCosE()
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
	Initd          bool
	ainit, atarget float64
	iinit, itarget float64
	einit, etarget float64
	Ωinit, Ωtarget float64
	ωinit, ωtarget float64
	controls       []ThrustControl
	GenericCL
}

// NewOptimalΔOrbit generates a new OptimalΔOrbit based on the provided target orbit.
func NewOptimalΔOrbit(target Orbit, laws ...ControlLaw) *OptimalΔOrbit {
	cl := OptimalΔOrbit{}
	cl.atarget = target.a
	cl.etarget = target.e
	cl.itarget = target.i
	cl.ωtarget = target.ω
	cl.Ωtarget = target.Ω
	if len(laws) == 0 {
		laws = []ControlLaw{OptiΔaCL, OptigΔeCL, OptiΔiCL, OptiΔΩCL, OptiΔωCL}
	}
	cl.controls = make([]ThrustControl, len(laws))
	for i, law := range laws {
		cl.controls[i] = NewOptimalThrust(law, "multi-opti")
	}
	cl.GenericCL = GenericCL{"ΔOrbit", multiOpti}
	return &cl
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
		return thrust
	}

	factor := func(oscul, init, target, tol float64) float64 {
		if floats.EqualWithinAbs(init, target, tol) {
			return 1 // Don't want no NaNs now.
		}
		if floats.EqualWithinAbs(oscul, target, tol) {
			return 0
		}
		return (target - oscul) / math.Abs(target-init)
	}

	for _, ctrl := range cl.controls {
		var oscul, init, target, tol float64
		switch ctrl.Type() {
		case OptiΔaCL:
			oscul = o.a
			init = cl.ainit
			target = cl.atarget
			tol = distanceε
		case OptigΔeCL:
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
			tmpThrust := ctrl.Control(o)
			for i := 0; i < 3; i++ {
				thrust[i] += fact * tmpThrust[i]
			}
		}
	}
	return unit(thrust)
}
