package dataio

import (
	"encoding/json"
	"testing"
)

func TestBodyFrame(t *testing.T) {
	var input = []byte(`{"type":"Spice",
     "name":"CASSINI_SC_COORD"}`)
	var v = CgBodyFrame{}
	if err := json.Unmarshal(input, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "CASSINI_SC_COORD" || v.Type != "Spice" {
		t.Fatal("Incorrect name or type: ", v.String())
	}
}

func TestTrajectory(t *testing.T) {
	var input = []byte(`{
     "type": "InterpolatedStates",
     "source": "cassini-solstice.xyzv"
   }`)
	var v = CgTrajectory{}
	if err := json.Unmarshal(input, &v); err != nil {
		t.Fatal(err)
	}
	if v.Source != "cassini-solstice.xyzv" || v.Type != "InterpolatedStates" {
		t.Fatal("CgTrajectory: ", v.String())
	}
}

func TestGeometry(t *testing.T) {
	var data = []byte(`{
     "type":"Mesh",
     "meshRotation":[
        0,
        0,
        -0.70710677,
        0.70710677
     ],
     "size":0.005,
     "source":"models/cassini/Cassini_no_Huygens_03.3ds"
  }`)
	var v = CgGeometry{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	if v.Type != "Mesh" || v.Size != 0.005 || v.Source != "models/cassini/Cassini_no_Huygens_03.3ds" || len(v.Mesh) != 4 {
		t.Fatal("CgGeometry")
	}
}

func TestLabel(t *testing.T) {
	var data = []byte(`{
     "color":[
        0.6,
        1,
        1
     ],
     "fadeSize":1000000,
     "showText":true
  }`)
	var v = CgLabel{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}

	if v.FadeSize != 1000000 || v.ShowText != true || len(v.Color) != 3 || v.Color[0] != 0.6 || v.Color[1] != 1 || v.Color[2] != 1 {
		t.Fatal("CgLabel:", v.String())
	}
}

func TestTrajectoryPlot(t *testing.T) {
	var data = []byte(`{
     "color":[
        0.6,
        1,
        1
     ],
     "lineWidth":1,
     "duration":"14 d",
     "lead":"3 d",
     "fade":1,
     "sampleCount":100
  }`)
	var v = &CgTrajectoryPlot{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	if v.LineWidth != 1 || v.Duration != "14 d" || v.Lead != "3 d" || v.Fade != 1 || v.SampleCount != 100 {
		t.Fatal("CgTrajectoryPlot")
	}
}

// TestFullImport tests the full import of a JSON from the Cosmographia guide.
func TestFullImport(t *testing.T) {
	var input = []byte(`{
   "version":"1.0",
   "name":"Cosmographia CASSINI Example",
   "items":[
      {
         "class":"spacecraft",
         "name":"Cassini",
         "startTime":"1997-10-15 09:26:08.390 UTC",
         "endTime":"2017-09-29 23:58:00.000 UTC",
         "center":"Saturn",
         "trajectory":{
            "type": "InterpolatedStates",
            "source": "cassini-solstice.xyzv"
          },
         "bodyFrame":{
            "type":"Spice",
            "name":"CASSINI_SC_COORD"
         },
         "geometry":{
            "type":"Mesh",
            "meshRotation":[
               0,
               0,
               -0.70710677,
               0.70710677
            ],
            "size":0.005,
            "source":"models/cassini/Cassini_no_Huygens_03.3ds"
         },
         "label":{
            "color":[
               0.6,
               1,
               1
            ],
            "fadeSize":1000000,
            "showText":true
         },
         "trajectoryPlot":{
            "color":[
               0.6,
               1,
               1
            ],
            "lineWidth":1,
            "duration":"14 d",
            "lead":"3 d",
            "fade":1,
            "sampleCount":100
         }
      }
   ]
}`)
	var output = CgCatalog{}
	if err := json.Unmarshal(input, &output); err != nil {
		t.Fatal(err)
	}
	// Let's now test everything was loaded correctly.
	if output.Version != "1.0" || output.Name != "Cosmographia CASSINI Example" {
		t.Fatal("Version or Name are incorrect", output.String())
	}
	if len(output.Items) != 1 {
		t.Fatal("Found more than one Item.")
	}
	item := output.Items[0]
	if item.Center != "Saturn" || item.Class != "spacecraft" || item.Name != "Cassini" || item.StartTime != "1997-10-15 09:26:08.390 UTC" || item.EndTime != "2017-09-29 23:58:00.000 UTC" {
		t.Fatal("Item parameters are invalid.")
	}
	// Check structs are loaded and non nil.
	if item.Bodyframe == (&CgBodyFrame{}) || item.Geometry == (&CgGeometry{}) || item.Label == (&CgLabel{}) || item.Trajectory == (&CgTrajectory{}) || item.TrajectoryPlot == (&CgTrajectoryPlot{}) {
		t.Fatal("One or more structs are empty.")
	}
}

func TestInterpolatedStatesExport(t *testing.T) {
	records := []CgInterpolatedState{CgInterpolatedState{2441778.60122, []float64{-143540520.299, -42601828.5841, -2696.02946285}, []float64{7.0417278, -42.899928, -2.2465784}},
		CgInterpolatedState{2441778.60784, []float64{-143535384.971, -42625931.5103, -4127.97459159}, []float64{10.212578, -41.142538, -2.5831545}}}
	for i, record := range records {
		if i == 0 && record.ToText() != "2441778.601220 -143540520.299000 -42601828.584100 -2696.029463 7.041728 -42.899928 -2.246578" {
			t.Fatal("Failed at index 0.")
		} else if i == 1 && record.ToText() != "2441778.607840 -143535384.971000 -42625931.510300 -4127.974592 10.212578 -41.142538 -2.583155" {
			t.Fatal("Failed at index 1.")
		}
	}

}
