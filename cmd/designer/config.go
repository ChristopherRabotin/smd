package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

type commonPlanet struct {
	planet      smd.CelestialObject
	from, until time.Time
}

func (p commonPlanet) String() string {
	return fmt.Sprintf("%s: %s -> %s", p.planet.Name, p.from, p.until)
}

// Launch is the start planet
type Launch struct {
	commonPlanet
	maxC3 float64
}

func (l Launch) String() string {
	return l.commonPlanet.String() + fmt.Sprintf("\tC3: %.1f km^2/s^2", l.maxC3)
}

// Arrival is the end planet
type Arrival struct {
	commonPlanet
	maxVinf float64
}

func (l Arrival) String() string {
	return l.commonPlanet.String() + fmt.Sprintf("\tVinf: %.1f km^2/s^2", l.maxVinf)
}

// Flyby is an intermediatary planet
type Flyby struct {
	commonPlanet
	maxDeltaV          float64
	minPeriapsisRadius float64
}

func (f Flyby) String() string {
	return f.commonPlanet.String() + fmt.Sprintf("rP: %.1f km\tdeltaV: %.1f", f.minPeriapsisRadius, f.maxDeltaV)
}

func readAllFlybys() []Flyby {
	tmpFlybys := make([]Flyby, 20) // Unlikely to be more than 20.
	flybycount := 0
	for _, planetName := range viper.GetStringSlice("general.flybyplanets") {
		key := fmt.Sprintf("flyby.%s", planetName)
		planet, perr := smd.CelestialObjectFromString(strings.Replace(key, "flyby.", "", 1))
		if perr != nil {
			log.Fatalf("could not understand planet in `%s`", key)
		}
		flybycount++
		from, until := confReadFromUntil(key)
		maxDV := viper.GetFloat64(fmt.Sprintf("%s.deltaV", key))
		peri := viper.GetFloat64(fmt.Sprintf("%s.periapsis", key)) * planet.Radius
		position := viper.GetInt(fmt.Sprintf("%s.position", key))
		tmpFlybys[position] = Flyby{commonPlanet{planet, from, until}, maxDV, peri}
	}
	// Remove any empty item.
	flybys := make([]Flyby, flybycount)
	flybycount = 0
	for _, flyby := range tmpFlybys {
		if flyby != (Flyby{}) {
			flybys[flybycount] = flyby
			flybycount++
		}
	}
	return flybys
}

func readLaunch() Launch {
	key := "launch"
	from, until := confReadFromUntil(key)
	planet, perr := smd.CelestialObjectFromString(viper.GetString(fmt.Sprintf("%s.planet", key)))
	if perr != nil {
		log.Fatalf("could not understand planet in `%s`", key)
	}
	return Launch{commonPlanet{planet, from, until}, viper.GetFloat64(fmt.Sprintf("%s.maxC3", key))}
}

func readArrival() Arrival {
	key := "arrival"
	from, until := confReadFromUntil(key)
	planet, perr := smd.CelestialObjectFromString(viper.GetString(fmt.Sprintf("%s.planet", key)))
	if perr != nil {
		log.Fatalf("could not understand planet in `%s`", key)
	}
	return Arrival{commonPlanet{planet, from, until}, viper.GetFloat64(fmt.Sprintf("%s.maxVinf", key))}
}

func confReadFromUntil(mainKey string) (from, until time.Time) {
	fromKey := fmt.Sprintf("%s.from", mainKey)
	untilKey := fmt.Sprintf("%s.until", mainKey)
	return confReadJDEorTime(fromKey), confReadJDEorTime(untilKey)
}

func confReadJDEorTime(key string) (dt time.Time) {
	jde := viper.GetFloat64(key)
	if jde == 0 {
		var perr error
		dt, perr = time.Parse(dateTimeFormat, viper.GetString(key))
		if perr != nil {
			log.Fatalf("could not understand `%s`: %s", key, perr)
		}
	} else {
		dt = julian.JDToTime(jde)
	}
	return
}
