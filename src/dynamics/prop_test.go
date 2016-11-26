package dynamics

import (
	"testing"
)

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

func TestPPS1350(t *testing.T) {
	thruster := new(PPS1350)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
	assertPanic(t, func() {
		v, p := thruster.Min()
		thruster.Thrust(v-1, p-1)
	})
}

/*func TestHPHET12k5(t *testing.T) {
	thruster := new(HPHET12k5)
	thruster.Thrust(thruster.Min())
	thruster.Thrust(thruster.Max())
	assertPanic(t, func() {
		v, p := thruster.Min()
		thruster.Thrust(v-1, p-1)
	})
}
*/
