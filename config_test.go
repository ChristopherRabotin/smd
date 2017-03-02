package smd

import (
	"testing"
	"time"

	"github.com/gonum/floats"
)

func TestSpice(t *testing.T) {
	cfgLoaded = true
	config = _smdconfig{SPICE: true, SPICEDir: "./cmd/refframes"}
	orbit := Mars.HelioOrbit(time.Date(2015, 06, 20, 0, 0, 0, 0, time.UTC))
	expR := []float64{1.727827778754413e+07, 2.3260881157426503e+08, 4.4498675818615705e+06}
	expV := []float64{-23.242444259084987, 3.852356165072191, 0.651187192849382}
	if !floats.Equal(orbit.rVec, expR) {
		t.Fatal("incorrect R")
	}
	if !floats.Equal(orbit.vVec, expV) {
		t.Fatal("incorrect V")
	}
}
