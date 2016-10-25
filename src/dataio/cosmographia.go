package dataio

import (
	"encoding/csv"
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

// StreamInterpolatedStates streams the output of the channel to the provided file.
func StreamInterpolatedStates(filename string, histChan <-chan (*CgInterpolatedState), stamped bool) {
	if stamped {
		t := time.Now()
		filename = fmt.Sprintf("../outputdata/prop%s-%d-%02d-%02dT%02d.%02d.%02d.xyzv", filename, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	} else {
		filename = fmt.Sprintf("../outputdata/prop%s.xyzv", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Printf("Saving file to %s.\n", f.Name())
	// Header
	f.WriteString(fmt.Sprintf(`# Creation date (UTC): %s
# Records are <jd> <x> <y> <z> <vel x> <vel y> <vel z>
#   Time is a TDB Julian date
#   Position in km
#   Velocity in km/sec`, time.Now()))
	// Read from channel
	previousJD := 0.0
	for {
		state, more := <-histChan
		if more {
			// Only write one data point per julian minute.
			if state.JD-previousJD < 1.0/(24*60) {
				continue
			} else if previousJD == 0 {
				// First iteration, let's add the initial time in simulation.
				f.WriteString(fmt.Sprintf("\n# Simulation time start (UTC): %s", julian.JDToTime(state.JD).UTC()))
			}
			previousJD = state.JD
			_, err := f.WriteString("\n" + state.ToText())
			if err != nil {
				panic(err)
			}
		} else {
			f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", julian.JDToTime(previousJD).UTC()))
			return
		}
	}
}
