package dynamics

import "math"

// ControlLaw defines an enum of control laws.
type ControlLaw uint8

const (
	tangential ControlLaw = iota + 1
	antiTangential
	inversion
	coast
)

func (cl ControlLaw) String() string {
	switch cl {
	case tangential:
		return "tangential"
	case antiTangential:
		return "antiTangential"
	case inversion:
		return "inversion"
	case coast:
		return "coast"
	}
	panic("unknown control law")
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

// ThrustControl defines a thrust control interface.
type ThrustControl interface {
	Control(o Orbit) []float64
	Type() ControlLaw
	Reason() string
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
	return unit(o.V)
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
	unitV := unit(o.V)
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
	f := o.Getν()
	if _, e := o.GetE(); e > 0.01 || (f > cl.ν-math.Pi && f < math.Pi-cl.ν) {
		return Tangential{}.Control(o)
	}
	return AntiTangential{}.Control(o)
}
