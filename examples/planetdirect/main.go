package main

import (
	"fmt"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func sc2Mars(arrivalDT time.Time) *smd.Spacecraft {
	marsOrbit := smd.Mars.HelioOrbit(arrivalDT)
	distance := marsOrbit.RNorm()
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := 5000.0
	ref2Mars := &smd.WaypointAction{Type: smd.REFMARS, Cargo: nil}
	a, e := smd.Radii2ae(44500+smd.Mars.Radius, 426+smd.Mars.Radius)
	var i float64 = 61
	var Ω float64 = 240
	var ν float64 = 180
	hyper := smd.NewOrbitFromOE(a, e, i, Ω, 60, ν, smd.Mars)
	return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{
			smd.NewReachDistance(distance, true, ref2Mars),
			smd.NewLoiter(time.Hour, nil),
			smd.NewToElliptical(nil),
			smd.NewOrbitTarget(*hyper, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL),
			smd.NewLoiter(7*24*time.Hour, nil),
		})
}

func sc2Earth(fuel float64, arrivalDT time.Time) *smd.Spacecraft {
	distance := smd.Earth.HelioOrbit(arrivalDT).RNorm()
	eps := smd.NewUnlimitedEPS()
	thrusters := []smd.EPThruster{smd.NewGenericEP(5, 5000)} // VASIMR (approx.)
	dryMass := 10000.0
	fuelMass := fuel
	ref2Earth := &smd.WaypointAction{Type: smd.REFEARTH, Cargo: nil}
	a, e := smd.Radii2ae(39300+smd.Earth.Radius, 290+smd.Earth.Radius)
	// Uses the min *and* max values, since it only depends on the argument of periapsis.
	var i float64 = 31
	var Ω float64 = 330
	var ν float64 = 210
	hyper := smd.NewOrbitFromOE(a, e, i, Ω, 180, ν, smd.Mars)
	return smd.NewSpacecraft("d2m", dryMass, fuelMass, eps, thrusters, false, []*smd.Cargo{},
		[]smd.Waypoint{
			smd.NewReachDistance(distance+smd.Earth.SOI, false, ref2Earth),
			smd.NewLoiter(time.Hour, nil),
			smd.NewToElliptical(nil),
			smd.NewOrbitTarget(*hyper, nil, smd.Naasz, smd.OptiΔaCL, smd.OptiΔiCL),
			smd.NewLoiter(7*24*time.Hour, nil),
		})
}

func main() {
	depart := time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC) // earth departure date
	maxToMars := depart.Add(2 * 365 * 24 * time.Hour)
	arrival := time.Date(2018, 14, 8, 0, 0, 0, 0, time.UTC) // mars arrival date
	initOrbit := smd.Earth.HelioOrbit(depart)
	vehicle := sc2Mars(arrival)
	astro := smd.NewMission(vehicle, &initOrbit, depart, maxToMars, smd.Perturbations{}, false, smd.ExportConfig{Filename: "d2m", AsCSV: false, Cosmo: true, Timestamp: false})
	astro.Propagate()
	R, V := initOrbit.RV()
	fmt.Printf("%+v\n%+v\n", R, V)
	/*fmt.Println("=== RETURN TRIP ===")
	// Now perform the return trip
	expectedArrival := astro.CurrentDT.Add(time.Duration(6*31*24) * time.Hour) // mars arrival date
	earthDistance := smd.Earth.HelioOrbit(expectedArrival).RNorm()
	vehicle.WayPoints = []smd.Waypoint{smd.NewReachDistance(earthDistance+smd.Earth.SOI, false, nil)}
	astro = smd.NewMission(vehicle, &initOrbit, astro.CurrentDT, depart.Add(-1), smd.Perturbations{}, false, smd.ExportConfig{Filename: "d2e", AsCSV: false, Cosmo: true, Timestamp: false})
	astro.Propagate()*/
}
