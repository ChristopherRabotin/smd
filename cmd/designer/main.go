package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

const (
	defaultScenario = "~~unset~~"
	dateTimeFormat  = "2006-01-02 15:04:05"
)

var (
	scenario string
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if scenario == defaultScenario {
		log.Fatal("no scenario provided and no finder set")
	}
	// Load scenario
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("./%s.toml not found", scenario)
	}
	// Read scenario
	prefix := viper.GetString("General.fileprefix")
	verbose := viper.GetBool("General.verbose")
	if verbose {
		fmt.Printf("[info] file prefix: %s\n", prefix)
	}
	timeStepStr := viper.GetString("General.step")
	timeStep, durErr := time.ParseDuration(timeStepStr)
	if durErr != nil {
		log.Fatalf("could not understand `step`: %s", durErr)
	}
	if verbose {
		fmt.Printf("[info] time step: %s\n", timeStep)
	}
	// Date time information
	var initLaunch, maxArrival time.Time
	var perr error
	initLaunchJD := viper.GetFloat64("General.from")
	if initLaunchJD == 0 {
		initLaunch, perr = time.Parse(dateTimeFormat, viper.GetString("General.from"))
		if perr != nil {
			log.Fatalf("could not understand `from`: %s", perr)
		}
	} else {
		initLaunch = julian.JDToTime(initLaunchJD)
	}
	if verbose {
		fmt.Printf("[info] init launch: %s\n", initLaunch)
	}
	maxArrivalJD := viper.GetFloat64("General.until")
	if maxArrivalJD == 0 {
		maxArrival, perr = time.Parse(dateTimeFormat, viper.GetString("General.until"))
		if perr != nil {
			log.Fatalf("could not understand `until`: %s", perr)
		}
	} else {
		maxArrival = julian.JDToTime(maxArrivalJD)
	}
	if verbose {
		fmt.Printf("[info] max arrival: %s\n", maxArrival)
	}
	// Read all the planets.
	planets := []smd.CelestialObject{}
	for pNo, planetStr := range viper.GetStringSlice("General.planets") {
		planet, err := smd.CelestialObjectFromString(planetStr)
		if err != nil {
			log.Fatalf("could not read planet %d: %s", pNo, err)
		}
		if verbose {
			fmt.Printf("[info] #%d: %s\n", pNo, planet.Name)
		}
		planets = append(planets, planet)
	}
}
