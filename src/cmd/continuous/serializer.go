package main

import (
	"dataio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Serializes the data to a file.

// CGOut generates the JSON file of the export.
func CGOut(name string, startDT, endDT time.Time) {
	if endDT.Before(startDT) {
		endDT = startDT.Add(time.Duration(24*100) * time.Hour)
	}
	traj := dataio.CgTrajectory{Type: "InterpolatedStates", Source: "prop" + name + ".xyzv"}
	label := dataio.CgLabel{Color: []float64{0.6, 1, 1}, FadeSize: 1000000, ShowText: true}
	plot := dataio.CgTrajectoryPlot{Color: []float64{0.6, 1, 1}, LineWidth: 1, Duration: "200 d", Lead: "0 d", Fade: 0, SampleCount: 10}
	// TODO: Split up the plots based on what part of the orbit is being propagated.
	item := dataio.CgItems{Class: "spacecraft", Name: name, StartTime: fmt.Sprintf("%s", startDT.UTC()), EndTime: fmt.Sprintf("%s", endDT.UTC()), Center: "Earth", Trajectory: &traj, Bodyframe: nil, Geometry: nil, Label: &label, TrajectoryPlot: &plot}
	c := dataio.CgCatalog{Version: "1.0", Name: name, Items: []*dataio.CgItems{&item}, Require: nil}
	// Create JSON file.
	f, err := os.Create("../outputdata/catalog-" + name + ".json")
	fmt.Printf("Saving file to %s.\n", f.Name())
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if marsh, err := json.Marshal(c); err != nil {
		panic(err)
	} else {
		f.Write(marsh)
	}
}
