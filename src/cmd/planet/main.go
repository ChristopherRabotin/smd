package main

import (
	"dynamics"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/soniakeys/meeus/julian"
)

// createInterpolatedFile returns a file which requires a defer close statement!
func createInterpolatedFile(filename string, stateDT time.Time) *os.File {
	filename = fmt.Sprintf("%s/prop-%s.xyzv", os.Getenv("DATAOUT"), filename)

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	// Header
	f.WriteString(fmt.Sprintf(`# Creation date (UTC): %s
# Records are <jd> <x> <y> <z> <vel x> <vel y> <vel z>
#   Time is a TDB Julian date
#   Position in km
#   Velocity in km/sec
#   Simulation time start (UTC): %s`, time.Now(), stateDT.UTC()))
	return f
}

func main() {
	filename := "Ear2"
	fileNo := 0
	dt := time.Now().UTC()
	start := time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	f := createInterpolatedFile(fmt.Sprintf("%s-%d", filename, fileNo), dt)
	defer f.Close()
	traj := dynamics.CgTrajectory{Type: "InterpolatedStates", Source: fmt.Sprintf("prop-%s-%d.xyzv", filename, fileNo)}
	// TODO: Switch color based on SC state (e.g. no fuel, not thrusting, etc.)
	label := dynamics.CgLabel{Color: []float64{1, 0.6, 1}, FadeSize: 1000000, ShowText: true}
	plot := dynamics.CgTrajectoryPlot{Color: []float64{0.6, 1, 1}, LineWidth: 1, Duration: "", Lead: "0 d", Fade: 0, SampleCount: 10}
	curCgItem := &dynamics.CgItems{Class: "spacecraft", Name: filename, StartTime: start.String(), EndTime: "", Center: "Sun", Trajectory: &traj, Bodyframe: nil, Geometry: nil, Label: &label, TrajectoryPlot: &plot}
	// Propagate the Earth.
	for start.Before(end) {
		orb := dynamics.Earth.HelioOrbit(start)
		R, V := orb.GetRV()
		is := dynamics.CgInterpolatedState{JD: julian.TimeToJD(start), Position: R, Velocity: V}
		if _, err := f.WriteString("\n" + is.ToText()); err != nil {
			panic(err)
		}
		start = start.Add(time.Duration(1) * time.Minute)
	}

	f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", start.UTC()))
	f.Close()
	longerEnd := start.Add(time.Duration(1) * time.Hour)
	curCgItem.EndTime = fmt.Sprintf("%s", longerEnd.UTC())
	curCgItem.TrajectoryPlot.Duration = "370 d"

	// Let's write the catalog.
	c := dynamics.CgCatalog{Version: "1.0", Name: filename, Items: []*dynamics.CgItems{curCgItem}, Require: nil}
	// Create JSON file.

	f, err := os.Create(os.Getenv("DATAOUT") + "/catalog-" + filename + ".json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Printf("Saving file to %s.\n", f.Name())
	if marsh, err := json.Marshal(c); err != nil {
		panic(err)
	} else {
		f.Write(marsh)
	}
}
