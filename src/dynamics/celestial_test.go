package dynamics

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestCelestialObject(t *testing.T) {
	for _, object := range []CelestialObject{Sun, Earth, Mars} {
		if object.String() != fmt.Sprintf("[Object %s]", object.Name) {
			t.Fatalf("invalid String for %s", object.Name)
		}
		object.HelioOrbit(time.Now().UTC())
	}
}

func TestPanics(t *testing.T) {
	assertPanic(t, func() {
		fake := CelestialObject{"Fake", -1, -1, -1, -1, -1, -1, nil}
		fake.HelioOrbit(time.Now())
	})
	assertPanic(t, func() {
		venus := CelestialObject{"Venus", -1, -1, -1, -1, -1, -1, nil}
		venus.HelioOrbit(time.Now())
	})
}

func TestHelio(t *testing.T) {
	dt := time.Date(2017, 03, 20, 14, 45, 0, 0, time.UTC)
	hR1, hV1 := Earth.HelioOrbit(dt)
	hR2, hV2 := Earth.HelioOrbit(dt.Add(time.Duration(1) * time.Minute))
	if math.Abs(norm(hR1)-norm(hR2)) > 1e2 {
		t.Fatal("radius changed by more than 100 km in a minute")
	}
	if math.Abs(norm(hV1)-norm(hV2)) > 1e-4 {
		t.Fatal("velocity changed by more than 1 m/s in a minute")
	}
}
