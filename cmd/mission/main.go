package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
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
	verbose  bool
	wg       sync.WaitGroup
)

func init() {
	// Read flags
	flag.StringVar(&scenario, "scenario", defaultScenario, "designer scenario TOML file")
	flag.BoolVar(&verbose, "verbose", false, "really verbose (esp. for configuration)")
}

// This is a LONG main function but it's because all the variables are used specifically to start the mission.
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
	timeStep := viper.GetDuration("mission.step")

	// Read spacecraft
	scName := viper.GetString("spacecraft.name")
	fuelMass := viper.GetFloat64("spacecraft.fuel")
	dryMass := viper.GetFloat64("spacecraft.dry")
	sc := smd.NewSpacecraft(scName, dryMass, fuelMass, smd.NewUnlimitedEPS(), []smd.EPThruster{}, true, []*smd.Cargo{}, []smd.Waypoint{})

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

	// Read randomness
	if probability := viper.GetFloat64("error.probability"); probability > 0 {
		position := viper.GetFloat64("error.position")
		velocity := viper.GetFloat64("error.velocity")
		perts.Noise = smd.NewOrbitNoise(probability, position, velocity)
	}

	// Maneuvers
	for burnNo := 0; viper.IsSet(fmt.Sprintf("burns.%d", burnNo)); burnNo++ {
		burnDT := confReadJDEorTime(fmt.Sprintf("burns.%d.date", burnNo))
		R := viper.GetFloat64(fmt.Sprintf("burns.%d.R", burnNo))
		N := viper.GetFloat64(fmt.Sprintf("burns.%d.N", burnNo))
		C := viper.GetFloat64(fmt.Sprintf("burns.%d.C", burnNo))
		sc.Maneuvers[burnDT] = smd.NewManeuver(R, N, C)
		if burnDT.After(endDT) || burnDT.Before(startDT) {
			log.Printf("[WARNING] burn scheduled out of propagation time")
		} else if verbose {
			log.Printf("Scheduled burn %s @ %s", sc.Maneuvers[burnDT], burnDT)
		}
	}

	exportConf := smd.ExportConfig{AsCSV: false, Cosmo: true, Filename: scName}
	mission := smd.NewPreciseMission(sc, scOrbit, startDT, endDT, perts, timeStep, false, exportConf)

	// Stations
	if viper.GetBool("measurements.enabled") {
		// Read stations
		stationNames := viper.GetStringSlice("measurements.stations")
		stations := make([]smd.Station, len(stationNames))
		for stNo, stationName := range stationNames {
			if len(stationName) > 8 && stationName[0:8] == "builtin." {
				stations[stNo] = smd.BuiltinStationFromName(stationName[8:len(stationName)])
			} else {
				// Read provided station.
				stationKey := fmt.Sprintf("station.%s.", stationName)
				humanName := viper.GetString(stationKey + "name")
				altitude := viper.GetFloat64(stationKey + "altitude")
				elevation := viper.GetFloat64(stationKey + "elevation")
				latΦ := viper.GetFloat64(stationKey + "latitude")
				longθ := viper.GetFloat64(stationKey + "longitude")
				σρ := viper.GetFloat64(stationKey + "range_sigma")
				σρDot := viper.GetFloat64(stationKey + "rate_sigma")
				st := smd.NewStation(humanName, altitude, elevation, latΦ, longθ, σρ, σρDot)
				if planetName := viper.GetString(stationKey + "planet"); len(planetName) > 0 {
					// A planet was specified, so it might not be Earth
					if planet, err := smd.CelestialObjectFromString(planetName); err != nil {
						log.Fatalf("could not use `%s` as planet for station `%s`: %s", planetName, humanName, err)
					} else {
						st.Planet = planet
					}
				}
				stations[stNo] = st
			}
			log.Printf("[info] added station %s", stations[stNo])
		}

		measChan := make(chan (smd.State))
		mission.RegisterStateChan(measChan)
		wg.Add(1)

		go func() {
			// Create measurement file
			f, err := os.Create(viper.GetString("measurements.output"))
			if err != nil {
				panic(fmt.Errorf("error creating file `%s`: %s", viper.GetString("measurements.output"), err))
			}
			// Header
			f.WriteString(fmt.Sprintf("# Creation date (UTC): %s\n\"station name\",\"epoch UTC\",\"Julian day\",\"Theta GST\",\"range (km)\",\"range rate (km/s)\"\n", time.Now()))
			// Iterate over each state
			numVis := 0
			for {
				state, more := <-measChan
				if !more {
					break
				}
				Δt := state.DT.Sub(startDT).Seconds()
				θgst := Δt * scOrbit.Origin.RotRate
				for _, st := range stations {
					measurement := st.PerformMeasurement(θgst, state)
					if measurement.Visible {
						f.WriteString(fmt.Sprintf("\"%s\",\"%s\",%f,%f,%s\n", st.Name, state.DT.Format(dateFormat), julian.TimeToJD(state.DT), θgst, measurement.ShortCSV()))
						numVis++
					}
				}
			}
			f.Close()
			log.Printf("[info] Generated %d measurements", numVis)
			wg.Done()
		}()
	}

	mission.PropagateUntil(endDT, true)
	wg.Wait()
}

func confReadJDEorTime(key string) (dt time.Time) {
	jde := viper.GetFloat64(key)
	if jde == 0 {
		dt = viper.GetTime(key)
	} else {
		dt = julian.JDToTime(jde)
	}
	if dt == (time.Time{}) {
		log.Fatalf("[error] could not parse date time in `%s`", key)
	}
	return
}
