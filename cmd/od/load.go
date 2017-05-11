package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

func loadMeasurementFile(filename string, stations map[string]smd.Station, ordering map[string]int) (map[time.Time][]smd.Measurement, []time.Time) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	cnt := 0
	measurements := make(map[time.Time][]smd.Measurement)
	measurementTimes := make([]time.Time, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Remove double quotes
		line = strings.Replace(line, "\"", "", -1)
		if len(line) == 0 || line[0:1] == "#" {
			continue
		}
		if cnt == 0 { // Skip header line
			cnt++
			continue
		}
		// "DSS34Canberra","2015-02-03 01:56:00 +0000 UTC",2457056.580556,107762.148457,16.715366,
		entries := strings.Split(line, ",")
		stationName := entries[0]
		// Check that the station exists, and complain otherwise.
		station, stExists := stations[stationName]
		if !stExists {
			log.Printf("[WARNING] skipping unknown station `%s` in measurement file\n", stationName)
			continue
		}
		stateDT, perr := time.Parse(dateFormat, entries[1])
		if perr != nil {
			log.Printf("[WARNING] skipping malformatted date `%s` in measurement file: %s\n", entries[1], perr)
			continue
		}
		Timeθgst, ferr := strconv.ParseFloat(entries[3], 64)
		if ferr != nil {
			log.Printf("[WARNING] skipping malformatted θgst `%s` in measurement file: %s\n", entries[3], ferr)
			continue
		}
		stRange, ferr0 := strconv.ParseFloat(entries[4], 64)
		if ferr0 != nil {
			log.Printf("[WARNING] skipping malformatted range `%s` in measurement file: %s\n", entries[4], ferr0)
			continue
		}
		stRate, ferr1 := strconv.ParseFloat(entries[5], 64)
		if ferr1 != nil {
			log.Printf("[WARNING] skipping malformatted raneg rate `%s` in measurement file: %s\n", entries[5], ferr1)
			continue
		}
		measurementTimes = append(measurementTimes, stateDT)
		measurement := smd.Measurement{Visible: true, Range: stRange, RangeRate: stRate, Timeθgst: Timeθgst, State: smd.State{DT: stateDT}, Station: station}
		if _, exists := measurements[stateDT]; !exists {
			measurements[stateDT] = make([]smd.Measurement, len(stations))
		}
		// Set the measurement in the right position
		measurements[stateDT][ordering[measurement.Station.Name]] = measurement
		cnt++
	}
	return measurements, measurementTimes
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
