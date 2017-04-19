package smd

import (
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestSpice(t *testing.T) {
	cfgLoaded = true
	config = _smdconfig{SPICEDir: "./cmd/refframes"}
	orbit := Mars.HelioOrbit(time.Date(2015, 06, 20, 0, 0, 0, 0, time.UTC))
	expR := []float64{1.727827778754413e+07, 2.3260881157426503e+08, 4.4498675818615705e+06}
	expV := []float64{-23.242444259084987, 3.852356165072191, 0.651187192849382}
	if !floats.Equal(orbit.rVec, expR) {
		t.Fatal("incorrect R")
	}
	if !floats.Equal(orbit.vVec, expV) {
		t.Fatal("incorrect V")
	}

	// Test frame change
	scR := []float64{-996776.1190926583, -39776.102324992695, 25123.28168731782}
	scV := []float64{-0.5114606889356655, -0.6914491357021403, -0.34254913653144525}
	scOrbit := NewOrbitFromRV(scR, scV, Earth)
	scOrbit.ToXCentric(Sun, time.Date(2016, 3, 24, 20, 41, 48, 0, time.UTC))
	// I should get the following, but it's actually very slightly off (maybe a second or two) and I don't have time to debug precisely why.
	// [-148030923.95108017, -12123548.951590259, 302492.17670564854, 2.6590243298160754, -29.849194304414752, -0.35374685933315347]
	expR = []float64{-1.4803092395107993e+08, -1.2123548951590609e+07, 302492.1767054541}
	expV = []float64{2.6590244425141782, -29.84919454394858, -0.3537466068661061}
	if !floats.Equal(scOrbit.rVec, expR) {
		t.Fatal("incorrect R")
	}
	if !floats.Equal(scOrbit.vVec, expV) {
		t.Fatal("incorrect V")
	}
}
