package main

import (
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
)

const (
	r2d = 180 / math.Pi
	d2r = 1 / r2d
)

var (
	minRadius = 300 + smd.Earth.Radius // km
	launch    = time.Date(1989, 10, 8, 0, 0, 0, 0, time.UTC)
	vga       = time.Date(1990, 2, 10, 0, 0, 0, 0, time.UTC)
	ega1      = time.Date(1990, 12, 10, 0, 0, 0, 0, time.UTC)
	ega2      = time.Date(1992, 12, 9, 12, 0, 0, 0, time.UTC)
	joi       = time.Date(1996, 3, 21, 12, 0, 0, 0, time.UTC)
)
