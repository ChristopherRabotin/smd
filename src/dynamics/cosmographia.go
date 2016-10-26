package dynamics

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/soniakeys/meeus/julian"
)

// CgCatalog definiton.
type CgCatalog struct {
	Version string     `json:"version"`
	Name    string     `json:"name"`
	Items   []*CgItems `json:"items"`
	Require []string   `json:"require,omitempty"`
}

func (c *CgCatalog) String() string {
	return c.Name + "(" + c.Version + ")"
}

// CgItems definiton.
type CgItems struct {
	Class          string            `json:"class"`
	Name           string            `json:"name"`
	StartTime      string            `json:"startTime"`
	EndTime        string            `json:"endTime"`
	Center         string            `json:"center"`
	Trajectory     *CgTrajectory     `json:"trajectory,omitempty"`
	Bodyframe      *CgBodyFrame      `json:"bodyFrame,omitempty"`
	Geometry       *CgGeometry       `json:"geometry,omitempty"`
	Label          *CgLabel          `json:"label,omitempty"`
	TrajectoryPlot *CgTrajectoryPlot `json:"trajectoryPlot,omitempty"`
}

// CgTrajectory definition.
type CgTrajectory struct {
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
}

// Validate validates a CgTrajectory.
func (t *CgTrajectory) Validate() error {
	if t.Type != "InterpolatedStates" || !strings.HasSuffix(t.Source, "xyzv") {
		return errors.New("Only InterpolatedStates are currently supported in Cosmographia trajectory types.")
	}
	return nil
}

func (t *CgTrajectory) String() string {
	return t.Source + " as " + t.Type
}

// CgBodyFrame definiton.
type CgBodyFrame struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

func (c *CgBodyFrame) String() string {
	return c.Name + " (type: " + c.Type + ")"
}

// CgGeometry definiton.
type CgGeometry struct {
	Type   string    `json:"type,omitempty"`
	Mesh   []float64 `json:"meshRotation,omitempty"`
	Size   float64   `json:"size,omitempty"`
	Source string    `json:"source,omitempty"`
}

// CgLabel definiton.
type CgLabel struct {
	Color    []float64 `json:"color,omitempty"`
	FadeSize int       `json:"fadeSize,omitempty"`
	ShowText bool      `json:"showText,omitempty"`
}

func (l *CgLabel) String() string {
	return fmt.Sprintf("color %v, fade %d, show %v", l.Color, l.FadeSize, l.ShowText)
}

// CgTrajectoryPlot definition.
type CgTrajectoryPlot struct {
	Color       []float64 `json:"color,omitempty"`
	LineWidth   int       `json:"lineWidth,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Lead        string    `json:"lead,omitempty"`
	Fade        int       `json:"fade,omitempty"`
	SampleCount int       `json:"sampleCount,omitempty"`
}

// CgInterpolatedState definiton.
type CgInterpolatedState struct {
	JD       float64
	Position []float64
	Velocity []float64
}

// FromText initializes from text.
// The `record` parameter must be an array of seven items.
func (i *CgInterpolatedState) FromText(record []string) {
	if val, err := strconv.ParseFloat(record[0], 64); err != nil {
		panic(err)
	} else {
		i.JD = val
	}

	if posX, err := strconv.ParseFloat(record[1], 64); err != nil {
		panic(err)
	} else if posY, err := strconv.ParseFloat(record[2], 64); err != nil {
		panic(err)
	} else if posZ, err := strconv.ParseFloat(record[3], 64); err != nil {
		panic(err)
	} else {
		i.Position = []float64{posX, posY, posZ}
	}

	if velX, err := strconv.ParseFloat(record[4], 64); err != nil {
		panic(err)
	} else if velY, err := strconv.ParseFloat(record[5], 64); err != nil {
		panic(err)
	} else if velZ, err := strconv.ParseFloat(record[6], 64); err != nil {
		panic(err)
	} else {
		i.Velocity = []float64{velX, velY, velZ}
	}
}

// ToText converts to text for written output.
func (i *CgInterpolatedState) ToText() string {
	return fmt.Sprintf("%f %f %f %f %f %f %f", i.JD, i.Position[0], i.Position[1], i.Position[2], i.Velocity[0], i.Velocity[1], i.Velocity[2])
}

// ParseInterpolatedStates takes a string and converts that into a CgInterpolatedState.
func ParseInterpolatedStates(s string) []*CgInterpolatedState {
	var states = []*CgInterpolatedState{}
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = ' '
	r.Comment = '#'
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		state := CgInterpolatedState{}
		state.FromText(record)
		states = append(states, &state)
	}

	return states
}

// createInterpolatedFile returns a file which requires a defer close statement!
func createInterpolatedFile(filename string, stamped bool, stateDT time.Time) *os.File {
	if stamped {
		t := time.Now()
		filename = fmt.Sprintf("%s/prop-%s-%d-%02d-%02dT%02d.%02d.%02d.xyzv", os.Getenv("DATAOUT"), filename, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	} else {
		filename = fmt.Sprintf("%s/prop-%s.xyzv", os.Getenv("DATAOUT"), filename)
	}
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

// StreamStates streams the output of the channel to the provided file.
func StreamStates(filename string, stateChan <-chan (AstroState), stamped bool) {
	// Read from channel
	var prevStatePtr, firstStatePtr *AstroState
	var fileNo uint8
	var f *os.File
	fileNo = 0
	cgItems := []*CgItems{}
	var curCgItem CgItems
	for {
		state, more := <-stateChan
		if more {
			// Determine whether a new Cosmographia interpolated state file is needed.
			if prevStatePtr == nil {
				firstStatePtr = &state
				f = createInterpolatedFile(fmt.Sprintf("%s-%d", filename, fileNo), stamped, state.dt)
				fileNo++
				traj := CgTrajectory{Type: "InterpolatedStates", Source: fmt.Sprintf("prop-%s-%d.xyzv", filename, fileNo)}
				// TODO: Switch color based on SC state (e.g. no fuel, not thrusting, etc.)
				label := CgLabel{Color: []float64{0.6, 1, 1}, FadeSize: 1000000, ShowText: true}
				plot := CgTrajectoryPlot{Color: []float64{0.6, 1, 1}, LineWidth: 1, Duration: "", Lead: "0 d", Fade: 0, SampleCount: 10}
				curCgItem = CgItems{Class: "spacecraft", Name: fmt.Sprintf("%s-%d", state.sc.Name, fileNo), StartTime: fmt.Sprintf("%s", state.dt.UTC()), EndTime: "", Center: state.orbit.Origin.Name, Trajectory: &traj, Bodyframe: nil, Geometry: nil, Label: &label, TrajectoryPlot: &plot}
			} else {
				if !prevStatePtr.orbit.Origin.Equals(state.orbit.Origin) {
					// Before switching files, let's write the end of simulation time.
					f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", prevStatePtr.dt.UTC()))
					// Update plot time propagation.
					curCgItem.EndTime = fmt.Sprintf("%s", prevStatePtr.dt.UTC())
					fmt.Printf("duration %s\n", prevStatePtr.dt.Sub(firstStatePtr.dt))
					curCgItem.TrajectoryPlot.Duration = fmt.Sprintf("%d d", int(prevStatePtr.dt.Sub(firstStatePtr.dt).Hours()/24+1))
					// Add this CG item to the list of items.
					cgItems = append(cgItems, &curCgItem)
					// Switch files.
					f.Close()
					f = createInterpolatedFile(fmt.Sprintf("%s-%d", filename, fileNo), stamped, state.dt)
					fileNo++
					// Force writing this data point now instead of creating N new files.
					prevStatePtr = &state
					asTxt := CgInterpolatedState{JD: julian.TimeToJD(state.dt), Position: state.orbit.R, Velocity: state.orbit.V}
					_, err := f.WriteString("\n" + asTxt.ToText())
					if err != nil {
						panic(err)
					}
					continue
				}
			}
			// Only write one datapoint per minute.
			if prevStatePtr != nil && state.dt.Sub(prevStatePtr.dt) <= time.Duration(1)*time.Minute {
				continue
			}
			prevStatePtr = &state
			asTxt := CgInterpolatedState{JD: julian.TimeToJD(state.dt), Position: state.orbit.R, Velocity: state.orbit.V}
			_, err := f.WriteString("\n" + asTxt.ToText())
			if err != nil {
				panic(err)
			}
		} else {
			// The channel is closed, hence the simulation is over.
			f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", prevStatePtr.dt.UTC()))
			cgItems = append(cgItems, &curCgItem)
			break
		}
	}
	// Let's write the catalog.
	c := CgCatalog{Version: "1.0", Name: prevStatePtr.sc.Name, Items: cgItems, Require: nil}
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
