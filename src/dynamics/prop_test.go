package dynamics

import (
	"testing"
)

func TestPPS1350(t *testing.T) {
	thruster := new(PPS1350)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
}

func TestHPHET12k5(t *testing.T) {
	thruster := new(HPHET12k5)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
}
