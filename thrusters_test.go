package smd

import (
	"testing"
)

func TestTHPPS1350(t *testing.T) {
	thruster := new(PPS1350)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
	assertPanic(t, func() {
		v, p := thruster.Min()
		thruster.Thrust(v-1, p-1)
	})
}

func TestTHHERMeS(t *testing.T) {
	thruster := new(HERMeS)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
	assertPanic(t, func() {
		v, p := thruster.Min()
		thruster.Thrust(v-1, p-1)
	})
}

func TestTHGenericEP(t *testing.T) {
	thrust, isp := 1., 2.
	thruster := NewGenericEP(thrust, isp)
	thrust0, isp0 := thruster.Thrust(3, 4)
	thrust1, isp1 := thruster.Thrust(5, 6)
	if thrust != thrust0 || thrust != thrust1 {
		t.Fatal("invalid thrust returned")
	}
	if isp != isp0 || isp != isp1 {
		t.Fatal("invalid isp returned")
	}
}
