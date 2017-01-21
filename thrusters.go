package smd

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
		return 89e-3, 1650
	}
	panic("unsupported voltage or power provided")
}

// HERMeS is based on the NASA & Rocketdyne 12.5kW demo
type HERMeS struct{}

// Min implements the Thruster interface.
func (t *HERMeS) Min() (voltage, power uint) {
	return t.Max()
}

// Max implements the Thruster interface.
func (t *HERMeS) Max() (voltage, power uint) {
	return 800, 12500
}

// Thrust implements the Thruster interface.
func (t *HERMeS) Thrust(voltage, power uint) (thrust, isp float64) {
	if voltage == 800 && power == 12500 {
		return 0.680, 2960
	}
	panic("unsupported voltage or power provided")
}

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
