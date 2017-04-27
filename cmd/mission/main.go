package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

// This code effectively only read the configuration file and propagates the mission.

const (
	defaultScenario    = "~~unset~~"
	dateFormat         = "2006-01-02 15:04:05"
	dateFormatFilename = "2006-01-02-15.04.05"
)

var (
	scenario string
	timeStep time.Duration
	verbose  bool
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
	flag.BoolVar(&verbose, "verbose", false, "really verbose (esp. for configuration)")
}
func main() {
	flag.Parse()
	// Load scenario
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}
	scenario = strings.Replace(scenario, ".toml", "", 1)
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("./%s.toml: Error %s", scenario, err)
	}

	// Read Mission parameters
	startDT := confReadJDEorTime("mission.start")
	endDT := confReadJDEorTime("mission.end")
	timeStep = viper.GetDuration("mission.step")
	if verbose {
		log.Printf("[conf] time step: %s\n", timeStep)
	}

	// Read spacecraft
	scName := viper.GetString("spacecraft.name")
	fuelMass := viper.GetFloat64("spacecraft.fuel")
	dryMass := viper.GetFloat64("spacecraft.dry")
	sc := smd.NewSpacecraft(scName, dryMass, fuelMass, smd.NewUnlimitedEPS(), []smd.EPThruster{}, true, []*smd.Cargo{}, []smd.Waypoint{})

	// TODO: Read error and randomness

	// Read orbit
	centralBodyName := viper.GetString("orbit.body")
	centralBody, err := smd.CelestialObjectFromString(centralBodyName)
	if err != nil {
		log.Fatalf("could not understand body `%s`: %s", centralBodyName, err)
	}
	a := viper.GetFloat64("orbit.sma")
	e := viper.GetFloat64("orbit.ecc")
	i := viper.GetFloat64("orbit.inc")
	Ω := viper.GetFloat64("orbit.RAAN")
	ω := viper.GetFloat64("orbit.argPeri")
	ν := viper.GetFloat64("orbit.tAnomaly")
	scOrbit := smd.NewOrbitFromOE(a, e, i, Ω, ω, ν, centralBody)

	// Read perturbations
	bodies := viper.GetStringSlice("perturbations.bodies")
	enableJ2 := viper.GetBool("perturbations.J2")
	enableJ3 := viper.GetBool("perturbations.J3")
	enableJ4 := viper.GetBool("perturbations.J4")
	var pertBody *smd.CelestialObject
	for _, body := range bodies {
		celObj, err := smd.CelestialObjectFromString(body)
		if err != nil {
			log.Fatalf("could not understand body `%s`: %s", body, err)
		}
		// XXX: This logic needs work after more bodies are allowed.
		if !celObj.Equals(smd.Sun) {
			log.Printf("body `%s` not yet supported, skipping it in perturbations", body)
		} else {
			pertBody = &celObj
		}
	}
	var jN uint8 = 0
	if enableJ4 {
		jN = 4
	} else if enableJ3 {
		jN = 3
	} else if enableJ2 {
		jN = 2
	}
	perts := smd.Perturbations{Jn: jN, PerturbingBody: pertBody}

	// Maneuvers
	for burnNo := 0; viper.IsSet(fmt.Sprintf("burns.%d", burnNo)); burnNo++ {
		burnDT := confReadJDEorTime(fmt.Sprintf("burns.%d.date", burnNo))
		V := viper.GetFloat64(fmt.Sprintf("burns.%d.V", burnNo))
		N := viper.GetFloat64(fmt.Sprintf("burns.%d.N", burnNo))
		C := viper.GetFloat64(fmt.Sprintf("burns.%d.C", burnNo))
		sc.Maneuvers[burnDT] = smd.NewManeuver(V, N, C)
		if burnDT.After(endDT) || burnDT.Before(startDT) {
			log.Printf("[WARNING] burn scheduled out of propagation time")
		} else if verbose {
			log.Printf("added: %s", sc.Maneuvers[burnDT])
		}
	}

	smd.NewMission(sc, scOrbit, startDT, endDT, perts, false, smd.ExportConfig{}).Propagate()
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
