package dynamics

import "testing"

func TestAngles(t *testing.T) {
	for i := 0.0; i < 360; i += 0.5 {
		if ok, err := floatEqual(i, rad2deg(deg2rad(i))); !ok {
			t.Fatalf("incorrect conversion for %3.2f, %s", i, err)
		}
	}
}
