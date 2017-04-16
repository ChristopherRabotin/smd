package main

import (
	"fmt"
	"log"
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
	return fmt.Sprintf("%s: %s -> %s", p.planet.Name, p.from.Format(dateFormat), p.until.Format(dateFormat))
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
	isResonant         bool
	resonance          float64
}

func (f Flyby) String() string {
	if f.isResonant {
		return f.commonPlanet.String() + fmt.Sprintf("\tres. %.1f:1\trP: %.1f km\tdeltaV: %.1f", f.resonance, f.minPeriapsisRadius, f.maxDeltaV)
	}
	return f.commonPlanet.String() + fmt.Sprintf("\trP: %.1f km\tdeltaV: %.1f", f.minPeriapsisRadius, f.maxDeltaV)
}

func readAllFlybys(minDT, maxDT time.Time) []Flyby {
	tmpFlybys := make([]Flyby, 20) // Unlikely to be more than 20.
	flybycount := 0
	prevPlanets := make(map[string]int)
	for _, planetName := range viper.GetStringSlice("general.flybyplanets") {
		num, isResonance := prevPlanets[planetName]
		key := fmt.Sprintf("flyby.%s", planetName)
		if isResonance {
			key = fmt.Sprintf("flyby.%s.%d", planetName, num+1)
			prevPlanets[planetName]++
		} else {
			prevPlanets[planetName] = 0
		}
		planet, perr := smd.CelestialObjectFromString(planetName)
		if perr != nil {
			log.Fatalf("could not understand planet in `%s`", key)
		}
		flybycount++
		from, until := confReadFromUntil(key)
		maxDV := viper.GetFloat64(fmt.Sprintf("%s.deltaV", key))
		peri := viper.GetFloat64(fmt.Sprintf("%s.periapsis", key)) * planet.Radius
		position := viper.GetInt(fmt.Sprintf("%s.position", key))
		isResonant := viper.GetBool(fmt.Sprintf("%s.resonant", key))
		resonance := viper.GetFloat64(fmt.Sprintf("%s.resonance", key))
		tmpFlybys[position] = Flyby{commonPlanet{planet, from, until}, maxDV, peri, isResonant, resonance}
	}
	// Remove any empty item.
	flybys := make([]Flyby, flybycount)
	flybycount = 0
	for _, flyby := range tmpFlybys {
		if flyby != (Flyby{}) {
			if flyby.from == (time.Time{}) {
				flyby.from = minDT
			}
			if flyby.until == (time.Time{}) {
				flyby.until = maxDT
			}
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
		dt = viper.GetTime(key)
	} else {
		dt = julian.JDToTime(jde)
	}
	return
}
