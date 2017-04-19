package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

const (
	defaultScenario = "~~unset~~"
	dtFormat        = "2006-01-02 15:04:05"
)

var (
	scenario string
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "scenario TOML to generate the PCP from")
}

func main() {
	// Read the configuration file.
	flag.Parse()
	if scenario == defaultScenario {
		log.Fatal("no scenario provided")
	}
	scenario = strings.Replace(scenario, ".toml", "", 1)
	// Load scenario
	viper.AddConfigPath(".")
	viper.SetConfigName(scenario)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("./%s.toml not found", scenario)
	}
	// Read general info
	verbose := viper.GetBool("General.verbose")
	c3plot := viper.GetBool("General.c3plot")
	ttype := smd.TransferTypeFromInt(viper.GetInt("General.transfer_type"))
	// Departure information
	var initLaunch, initArrival, maxLaunch, maxArrival time.Time
	var perr error
	// Ugh code duplication (but I'm in a hurry).
	initLaunchJD := viper.GetFloat64("Departure.from")
	if initLaunchJD == 0 {
		initLaunch, perr = time.Parse(dtFormat, viper.GetString("Departure.from"))
		if perr != nil {
			log.Print(perr)
			log.Fatal("could not read Departure.from")
		}
	} else {
		initLaunch = julian.JDToTime(initLaunchJD)
	}
	initArrivalJD := viper.GetFloat64("Arrival.from")
	if initArrivalJD == 0 {
		initArrival, perr = time.Parse(dtFormat, viper.GetString("Arrival.from"))
		if perr != nil {
			log.Fatal("could not read Arrival.from")
		}
	} else {
		initArrival = julian.JDToTime(initArrivalJD)
	}
	maxLaunchJD := viper.GetFloat64("Departure.until")
	if initLaunchJD == 0 {
		maxLaunch, perr = time.Parse(dtFormat, viper.GetString("Departure.until"))
		if perr != nil {
			log.Fatal("could not read Departure.until")
		}
	} else {
		maxLaunch = julian.JDToTime(maxLaunchJD)
	}
	maxArrivalJD := viper.GetFloat64("Arrival.until")
	if maxArrivalJD == 0 {
		maxArrival, perr = time.Parse(dtFormat, viper.GetString("Arrival.until"))
		if perr != nil {
			log.Fatal("could not read Arrival.until")
		}
	} else {
		maxArrival = julian.JDToTime(maxArrivalJD)
	}
	// Get the resolution
	resoInit := viper.GetFloat64("Departure.resolution")
	resoArr := viper.GetFloat64("Arrival.resolution")
	// Get planets
	initPlanet, plerr := smd.CelestialObjectFromString(viper.GetString("Departure.planet"))
	if plerr != nil {
		log.Fatal(plerr)
	}
	arrivalPlanet, plerr := smd.CelestialObjectFromString(viper.GetString("Arrival.planet"))
	if plerr != nil {
		log.Fatal(plerr)
	}
	smd.PCPGenerator(initPlanet, arrivalPlanet, initLaunch, maxLaunch, initArrival, maxArrival, resoInit, resoArr, ttype, c3plot, verbose, true)
	return
}
