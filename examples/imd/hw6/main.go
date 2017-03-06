package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

const (
	hwQ = 1
)

var (
	launch = time.Date(1989, 10, 8, 0, 0, 0, 0, time.UTC)
	vga1   = time.Date(1990, 2, 10, 0, 0, 0, 0, time.UTC)
	ega1   = time.Date(1990, 12, 10, 0, 0, 0, 0, time.UTC)
	ega2   = time.Date(1992, 12, 9, 12, 0, 0, 0, time.UTC)
	joi    = time.Date(1996, 3, 21, 12, 0, 0, 0, time.UTC)
)

func main() {
	fmt.Printf("%s\t~%f orbits\n", ega2.Sub(ega1), ega2.Sub(ega1).Hours()/(365.242189*24))
	fmt.Printf("==== QUESTION %d ====\n", hwQ)
	if hwQ == 1 {
		// hwQ 1
		vga1R := mat64.NewVector(3, smd.Venus.HelioOrbit(vga1).R())
		ega1R := mat64.NewVector(3, smd.Earth.HelioOrbit(ega1).R())
		_, Vf, _, _ := smd.Lambert(vga1R, ega1R, ega1.Sub(vga1), smd.TTypeAuto, smd.Sun)
		vfloats := []float64{Vf.At(0, 0), Vf.At(1, 0), Vf.At(2, 0)}
		egaOrbit := smd.NewOrbitFromRV(smd.Earth.HelioOrbit(ega1).R(), vfloats, smd.Sun)
		egaOrbit.ToXCentric(smd.Earth, ega1)
		fmt.Printf("%+v\n%f km/s\n", egaOrbit.V(), egaOrbit.VNorm())
		return
	}
	if hwQ == 2 {
		// hwQ 1
		ega2R := mat64.NewVector(3, smd.Earth.HelioOrbit(ega2).R())
		joiR := mat64.NewVector(3, smd.Jupiter.HelioOrbit(joi).R())
		Vi, _, _, _ := smd.Lambert(ega2R, joiR, joi.Sub(ega2), smd.TTypeAuto, smd.Sun)
		vfloats := []float64{Vi.At(0, 0), Vi.At(1, 0), Vi.At(2, 0)}
		egaOrbit := smd.NewOrbitFromRV(smd.Earth.HelioOrbit(ega2).R(), vfloats, smd.Sun)
		egaOrbit.ToXCentric(smd.Earth, ega2)
		fmt.Printf("%+v\n%f km/s\n", egaOrbit.V(), egaOrbit.VNorm())
		return
	}
}
