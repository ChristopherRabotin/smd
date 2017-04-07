package main

import (
	"flag"
	"log"
	"strconv"
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
		log.Printf("[info] file prefix: %s\n", prefix)
	}
	timeStepStr := viper.GetString("General.step")
	timeStep, durErr := time.ParseDuration(timeStepStr)
	if durErr != nil {
		log.Fatalf("could not understand `step`: %s", durErr)
	}
	if verbose {
		log.Printf("[info] time step: %s\n", timeStep)
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
		log.Printf("[info] init launch: %s\n", initLaunch)
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
		log.Printf("[info] max arrival: %s\n", maxArrival)
	}
	// Read all the planets.
	planets := []smd.CelestialObject{}
	for pNo, planetStr := range viper.GetStringSlice("General.planets") {
		planet, err := smd.CelestialObjectFromString(planetStr)
		if err != nil {
			log.Fatalf("could not read planet #%d: %s", pNo, err)
		}
		planets = append(planets, planet)
	}
	// Read and compute the radii constraints
	periapsisRadii := []float64{}
	for pNo, periRfactorStr := range viper.GetStringSlice("General.periRFactor") {
		periRfactor, err := strconv.ParseFloat(periRfactorStr, 64)
		if err != nil {
			log.Fatalf("could not read radius periapsis factor #%d: %s", pNo, err)
		}
		periapsisRadii = append(periapsisRadii, periRfactor*planets[pNo].Radius)
	}
	// Read the deltaV constraints
	maxDeltaVs := []float64{}
	for pNo, deltaVStr := range viper.GetStringSlice("General.maxDeltaV") {
		deltaV, err := strconv.ParseFloat(deltaVStr, 64)
		if err != nil {
			log.Fatalf("could not read maximum deltaV #%d: %s", pNo, err)
		}
		maxDeltaVs = append(maxDeltaVs, deltaV)
	}
	// Now summarize the planet passages
	if verbose {
		for pNo, planet := range planets {
			if pNo != len(planets)-1 {
				log.Printf("[info] #%d: %s\trP: %f km\tdeltaV: %f km/s\n", pNo, planet.Name, periapsisRadii[pNo], maxDeltaVs[pNo])
			} else {
				log.Printf("[info] #%d: %s (destination)\n", pNo, planet.Name)
			}
		}
	}
	// Read departure/arrival constraints.
	var maxC3, maxVinfArrival float64
	if viper.IsSet("DepartureConstraints.c3") {
		maxC3 = viper.GetFloat64("DepartureConstraints.c3")
	}
	if verbose {
		if maxC3 > 0 {
			log.Printf("[info] max c3: %f km^2/s^2\n", maxC3)
		} else {
			log.Println("[warn] no max c3 set")
		}
	}
	if viper.IsSet("ArrivalConstraints.vInf") {
		maxVinfArrival = viper.GetFloat64("ArrivalConstraints.vInf")
	}
	if verbose {
		if maxVinfArrival > 0 {
			log.Printf("[info] max vInf: %f km/s\n", maxVinfArrival)
		} else {
			log.Println("[warn] no max vInf set")
		}
	}
}
