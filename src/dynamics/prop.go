package dynamics

import (
	"fmt"
	"math"
)

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

// TangentialThrustCL thrusts in the tangenial direction at all times.
func TangentialThrustCL(o Orbit) []float64 {
	//return unit(o.V)
	return o.V
}

// InversionCL keeps the thrust as tangential but inverts its direction within an angle from the orbit apogee.
// cf. Izzo et al. (https://arxiv.org/pdf/1602.00849v2.pdf)
func InversionCL(o Orbit, ν float64) []float64 {
	f := o.GetΦ()
	unitV := unit(o.V)
	if f < ν-math.Pi || f > math.Pi-ν {
		fmt.Println("yo")
		unitV[0] *= -1
		unitV[1] *= -1
		unitV[2] *= -1
	}
	return unitV
}
