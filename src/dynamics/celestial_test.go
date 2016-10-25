package dynamics

import (
	"fmt"
	"testing"
	"time"
)

func TestCelestialObject(t *testing.T) {
	for _, object := range []CelestialObject{Sun, Earth, Mars} {
		if object.String() != fmt.Sprintf("[Object %s]", object.Name) {
			t.Fatalf("invalid String for %s", object.Name)
		}
		object.HelioOrbit(time.Now())
	}
}

func TestPanics(t *testing.T) {
	assertPanic(t, func() {
		fake := CelestialObject{"Fake", -1, -1, -1, -1, -1}
		fake.HelioOrbit(time.Now())
	})
	assertPanic(t, func() {
		venus := CelestialObject{"Venus", -1, -1, -1, -1, -1}
		venus.HelioOrbit(time.Now())
	})
}
