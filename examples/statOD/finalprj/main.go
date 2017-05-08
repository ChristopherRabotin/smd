package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

func main() {
	fmt.Printf("rA = %f km\trP = %f km\n", 101000+smd.Mars.Radius, 3919+smd.Mars.Radius)
	a, e := smd.Radii2ae(101000+smd.Mars.Radius, 3919+smd.Mars.Radius)
	tgo0 := smd.NewOrbitFromOE(a, e, 74, 0, 0, 0, smd.Mars)
	startDT := time.Date(2016, 10, 19, 0, 0, 0, 0, time.UTC)
	smd.NewMission(smd.NewEmptySC("name", 0), tgo0, startDT, startDT.Add(4*time.Hour), smd.Perturbations{}, false, smd.ExportConfig{}).Propagate()
	halfPeriod := time.Duration(tgo0.Period().Seconds()*0.5) * time.Second
	fmt.Printf("%s\nperiod = %s\thalf = %s\n", tgo0, tgo0.Period(), startDT.Add(-halfPeriod))
	Ri := mat64.NewVector(3, tgo0.R()) // Maneuver starts at periapsis
	minDV := math.Inf(1)
	var minν float64
	var minVi, minVf *mat64.Vector
	var minHours float64
	for hours := 0.1; hours < 10; hours += 0.05 {
		for ν := 0.0; ν < 360; ν += 0.5 {
			Rf := mat64.NewVector(3, smd.NewOrbitFromOE(400+smd.Mars.Radius, 0, 74, 0, 0, ν, smd.Mars).R())
			Vi, Vf, _, _ := smd.Lambert(Ri, Rf, time.Duration(hours)*time.Hour, smd.TTypeAuto, smd.Mars)
			if totalDV := mat64.Norm(Vi, 2) + mat64.Norm(Vf, 2); totalDV < minDV {
				minDV = totalDV
				minν = ν
				minHours = hours
				minVi = Vi
				minVf = Vf
			}
		}
	}
	fmt.Printf("ν = %f\t period: %f h\nbudget = %f km/s\nVi = %+v\nVf = %+v\n", minν, minHours, minDV, mat64.Formatted(minVi.T()), mat64.Formatted(minVf.T()))
}
