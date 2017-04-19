package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func loadMeasurementFile(version string, initDT time.Time) (map[time.Time]smd.Measurement, time.Time, time.Time) {
	file, err := os.Open(fmt.Sprintf("Project1%s_Obs.txt", version))
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var startDT, endDT time.Time
	cnt := 0
	measurements := make(map[time.Time]smd.Measurement)
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}
		if cnt == 0 { // Skip header line
			cnt++
			continue
		}
		entries := strings.Split(scanner.Text(), ",")
		epoch, err := strconv.ParseFloat(strings.TrimSpace(entries[0]), 64)
		if err != nil {
			log.Fatalf("[load:error] could not parse epoch when reading: `%s`", strings.TrimSpace(entries[0]))
		}
		stateDT := initDT.Add(time.Duration(epoch) * time.Second)
		ranges := make([]float64, 3)
		rates := make([]float64, 3)
		for i := 0; i < 3; i++ {
			tRg, err := strconv.ParseFloat(strings.TrimSpace(entries[i+1]), 64)
			if err != nil {
				log.Fatalf("[load:error] could not parse position when reading: `%s`", strings.TrimSpace(entries[i+1]))
			}
			if math.IsNaN(tRg) {
				continue
			}
			var station smd.Station
			switch i {
			case 0:
				station = _DSS34Canberra
			case 1:
				station = _DSS65Madrid
			default:
				station = _DSS13Goldstone
			}
			tRgR, err := strconv.ParseFloat(strings.TrimSpace(entries[i+4]), 64)
			if err != nil {
				log.Fatalf("[load:error] could not parse velocity when reading: `%s`", strings.TrimSpace(entries[i+4]))
			}
			ranges[i] = tRg
			rates[i] = tRgR
			measurements[stateDT] = smd.Measurement{Visible: true, Range: tRg, RangeRate: tRgR, TrueRange: tRg, TrueRangeRate: tRgR, Timeθgst: epoch * smd.EarthRotationRate2, State: smd.State{DT: stateDT}, Station: station}
		}
		if cnt == 1 {
			startDT = stateDT
		}
		endDT = stateDT
		cnt++
	}
	return measurements, startDT, endDT
}
