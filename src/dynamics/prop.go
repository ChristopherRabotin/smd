package dynamics

// Thruster defines a thruster interface.
type Thruster interface {
	// Returns the minimum power and voltage requirements for this thruster.
	Min() (voltage, power uint)
	// Returns the max power and voltage requirements for this thruster.
	Max() (voltage, power uint)
	// Returns the thrust in Newtons and the fuel mass consumed in kg.
	Thrust(voltage, power uint) (thrust, fuelMass float64)
}

/* Available thrusters */

// PPS1350 is the Snecma thruster used on SMART-1.
// Source: http://www.esa.int/esapub/bulletin/bulletin129/bul129e_estublier.pdf
type PPS1350 struct{}

// Min implements the Thruster interface.
func (t *PPS1350) Min() (voltage, power uint) {
	return 50, 1140
}

// Max implements the Thruster interface.
func (t *PPS1350) Max() (voltage, power uint) {
	return 50, 1140
}

// Thrust implements the Thruster interface.
func (t *PPS1350) Thrust(voltage, power uint) (thrust, fuelMass float64) {
	if voltage == 50 && power == 1140 {
		return 68 * 1e-3, 4.4 * 1e-6
	}
	panic("unsupported voltage or power provided")
}

// HPHET12k5 is based on the NASA & Rocketdyne 12.5kW demo
type HPHET12k5 struct{}

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
		return 0.680, 4.4 * 1e-5
	}
	panic("unsupported voltage or power provided")
}
