package dynamics

import (
	"errors"
	"log"
	"time"
)

// EPS defines the interface for an electrical power subsystem.
type EPS interface {
	Drain(voltage, power uint, dt time.Time) error
}

/* Available EPS */

// UnlimitedEPS drain as much as you want, always.
type UnlimitedEPS struct{}

// Drain implements the interface.
func (e *UnlimitedEPS) Drain(voltage, power uint, dt time.Time) error {
	return nil
}

// NewUnlimitedEPS returns a dream-like EPS.
func NewUnlimitedEPS() (e *UnlimitedEPS) {
	e = new(UnlimitedEPS)
	return
}

// TimedEPS sets a hard limit on how long (time-wise) the EPS can deliver any power.
type TimedEPS struct {
	turnedOn      bool          // Stores whether on or off.
	turnOnDT      time.Time     // Time at which it was turned on.
	turnOffDT     time.Time     // Time at which it was turned off.
	dischargeTime time.Duration // Max discharge duration.
	chargeTime    time.Duration // Charging duration.
}

// NewTimedEPS creates a new TimedEPS.
func NewTimedEPS(charge, discharge time.Duration) (t *TimedEPS) {
	t = new(TimedEPS)
	t.turnedOn = false
	t.turnOnDT = time.Now()
	t.turnOffDT = time.Now().Add(-charge) // Make the EPS available at start.
	t.dischargeTime = discharge
	t.chargeTime = charge
	return
}

// Drain implements the EPS subsystem.
func (t *TimedEPS) Drain(voltage, power uint, dt time.Time) error {
	if t.turnedOn {
		if dt.Sub(t.turnOnDT) >= t.dischargeTime {
			t.turnedOn = false
			t.turnOffDT = dt
			log.Println("EPS is off") // TODO: switch to context based logging...
			return errors.New("power now exhausted")
		}
		return nil
	}
	if dt.Sub(t.turnOffDT) >= t.chargeTime {
		t.turnedOn = true
		t.turnOnDT = dt
		log.Println("EPS is on")
		return nil
	}
	return errors.New("charging incomplete")

}
