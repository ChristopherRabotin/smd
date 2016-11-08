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

func TestFindEquinox(t *testing.T) {
	//t.SkipNow() // Equinox in dataset is 2017-03-20 14:45:00.
	dt := time.Date(2017, 01, 19, 0, 0, 0, 0, time.UTC)
	maxDt := time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC)
	equinox := time.Date(2010, 01, 01, 0, 0, 0, 0, time.UTC)
	// If the X is within 1% of one AU, then find the smallest Y.
	minY := 1e16
	// Finding the X which is the closest to
	for dt.Before(maxDt) {
		hR, _ := Earth.HelioOrbit(dt)
		if math.Abs(hR[0]-AU) < 0.1*AU {
			if diff := math.Abs(hR[1] - minY); minY > diff {
				minY = diff
				equinox = dt
			}
		}
		dt = dt.Add(time.Duration(1) * time.Minute)
	}
	hR, _ := Earth.HelioOrbit(equinox)
	t.Logf("Equinox: %+v - %+v", equinox, hR)
}

func TestHelio(t *testing.T) {
	dt := time.Date(2017, 03, 20, 14, 45, 0, 0, time.UTC)
	hR1, hV1 := Earth.HelioOrbit(dt)
	t.Logf("hR1 = %+v\n", hR1)
	hR2, hV2 := Earth.HelioOrbit(dt.Add(time.Duration(1) * time.Minute))
	if math.Abs(norm(hR1)-norm(hR2)) > 1e2 {
		t.Fatal("radius changed by more than 100 km in a minute")
	}
	if math.Abs(norm(hV1)-norm(hV2)) > 1e-4 {
		t.Fatal("velocity changed by more than 1 m/s in a minute")
	}
}
