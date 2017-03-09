package smd

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

// CgCatalog definition.
type CgCatalog struct {
	Version string     `json:"version"`
	Name    string     `json:"name"`
	Items   []*CgItems `json:"items"`
	Require []string   `json:"require,omitempty"`
}

func (c *CgCatalog) String() string {
	return c.Name + "(" + c.Version + ")"
}

// CgItems definition.
type CgItems struct {
	Class           string            `json:"class"`
	Name            string            `json:"name"`
	StartTime       string            `json:"startTime"`
	EndTime         string            `json:"endTime"`
	Center          string            `json:"center"`
	TrajectoryFrame string            `json:"trajectoryFrame"`
	Trajectory      *CgTrajectory     `json:"trajectory,omitempty"`
	Bodyframe       *CgBodyFrame      `json:"bodyFrame,omitempty"`
	Geometry        *CgGeometry       `json:"geometry,omitempty"`
	Label           *CgLabel          `json:"label,omitempty"`
	TrajectoryPlot  *CgTrajectoryPlot `json:"trajectoryPlot,omitempty"`
}

// CgTrajectory definition.
type CgTrajectory struct {
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
}

// Validate validates a CgTrajectory.
func (t *CgTrajectory) Validate() error {
	if t.Type != "InterpolatedStates" || !strings.HasSuffix(t.Source, "xyzv") {
		return errors.New("only InterpolatedStates are currently supported in Cosmographia trajectory types")
	}
	return nil
}

func (t *CgTrajectory) String() string {
	return t.Source + " as " + t.Type
}

// CgBodyFrame definition.
type CgBodyFrame struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

func (c *CgBodyFrame) String() string {
	return c.Name + " (type: " + c.Type + ")"
}

// CgGeometry definition.
type CgGeometry struct {
	Type   string    `json:"type,omitempty"`
	Mesh   []float64 `json:"meshRotation,omitempty"`
	Size   float64   `json:"size,omitempty"`
	Source string    `json:"source,omitempty"`
}

// CgLabel definition.
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

// CgInterpolatedState definition.
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
	config := smdConfig()
	if stamped {
		t := time.Now()
		filename = fmt.Sprintf("%s/prop-%s-%d-%02d-%02dT%02d.%02d.%02d.xyzv", config.outputDir, filename, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	} else {
		filename = fmt.Sprintf("%s/prop-%s.xyzv", config.outputDir, filename)
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

// createAsCSVCSVFile returns a file which requires a defer close statement!
func createAsCSVCSVFile(filename string, conf ExportConfig, stateDT time.Time) *os.File {
	config := smdConfig()
	if conf.Timestamp {
		t := time.Now()
		filename = fmt.Sprintf("%s/orbital-elements-%s-%d-%02d-%02dT%02d.%02d.%02d.csv", config.outputDir, filename, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	} else {
		filename = fmt.Sprintf("%s/orbital-elements-%s.csv", config.outputDir, filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	// Header
	f.WriteString(fmt.Sprintf(`# Creation date (UTC): %s
# Records are a, e, i, Ω, ω, ν. All angles are in degrees.
#   Simulation time start (UTC): %s
time,a,e,i,Omega,omega,nu,fuel,timeInHours,timeInDays,`, time.Now(), stateDT.UTC()))
	if conf.CSVAppendHdr != nil {
		// Append the headers for the appended columns.
		f.WriteString(conf.CSVAppendHdr())
	}
	return f
}

// StreamStates streams the output of the channel to the provided file.
func StreamStates(conf ExportConfig, stateChan <-chan (MissionState)) {
	// Read from channel
	var prevStatePtr, firstStatePtr *MissionState
	var fileNo uint8
	var f, fAsCSV *os.File
	fileNo = 0
	cgItems := []*CgItems{}
	var curCgItem *CgItems
	defer func() {
		if conf.Cosmo {
			// Let's write the catalog.
			c := CgCatalog{Version: "1.0", Name: prevStatePtr.SC.Name, Items: cgItems, Require: nil}
			// Create JSON file.

			fc, err := os.Create(fmt.Sprintf("%s/catalog-%s.json", smdConfig().outputDir, conf.Filename))
			if err != nil {
				panic(err)
			}
			defer fc.Close()
			fmt.Printf("Saving file to %s.\n", fc.Name())
			if marsh, err := json.Marshal(c); err != nil {
				panic(err)
			} else {
				fc.Write(marsh)
			}
		}
	}()

	color := []float64{0.6, 1, 1}
	for {
		state, more := <-stateChan
		if more {
			// Determine whether a new Cosmographia interpolated state file is needed.
			if prevStatePtr == nil {
				firstStatePtr = &state
				if conf.Cosmo {
					f = createInterpolatedFile(fmt.Sprintf("%s-%d", conf.Filename, fileNo), conf.Timestamp, state.DT)
					traj := CgTrajectory{Type: "InterpolatedStates", Source: fmt.Sprintf("prop-%s-%d.xyzv", conf.Filename, fileNo)}
					// TODO: Switch color based on SC state (e.g. no fuel, not thrusting, etc.)
					label := CgLabel{Color: color, FadeSize: 1000000, ShowText: true}
					plot := CgTrajectoryPlot{Color: color, LineWidth: 1, Duration: "", Lead: "0 d", Fade: 0, SampleCount: 10}
					curCgItem = &CgItems{Class: "spacecraft", Name: fmt.Sprintf("%s-%d", state.SC.Name, fileNo), StartTime: fmt.Sprintf("%s", state.DT.UTC()), EndTime: "", Center: state.Orbit.Origin.Name, Trajectory: &traj, Bodyframe: nil, Geometry: nil, Label: &label, TrajectoryPlot: &plot}
					if state.Orbit.Origin.Equals(Sun) {
						curCgItem.TrajectoryFrame = "EclipticJ2000"
					} else {
						curCgItem.TrajectoryFrame = "ICRF"
					}
				}
				if conf.AsCSV {
					fAsCSV = createAsCSVCSVFile(fmt.Sprintf("%s-%d", conf.Filename, fileNo), conf, state.DT)
				}
				fileNo++
			} else {
				if !prevStatePtr.Orbit.Origin.Equals(state.Orbit.Origin) {
					if conf.Cosmo {
						// Before switching files, let's write the end of simulation time.
						f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", state.DT.UTC()))
						// Update plot time propagation.
						longerEnd := state.DT.Add(time.Duration(1) * time.Hour)
						curCgItem.EndTime = fmt.Sprintf("%s", longerEnd.UTC())
						curCgItem.TrajectoryPlot.Duration = fmt.Sprintf("%d d", int(longerEnd.Sub(firstStatePtr.DT).Hours()/24+1))
						// Add this CG item to the list of items.
						cgItems = append(cgItems, curCgItem)
						// Switch files.
						f.Close()
						// XXX: Copy/paste from above :'(
						f = createInterpolatedFile(fmt.Sprintf("%s-%d", conf.Filename, fileNo), conf.Timestamp, state.DT)
						traj := CgTrajectory{Type: "InterpolatedStates", Source: fmt.Sprintf("prop-%s-%d.xyzv", conf.Filename, fileNo)}
						label := CgLabel{Color: color, FadeSize: 1000000, ShowText: true}
						plot := CgTrajectoryPlot{Color: color, LineWidth: 1, Duration: "", Lead: "0 d", Fade: 0, SampleCount: 10}
						curCgItem = &CgItems{Class: "spacecraft", Name: fmt.Sprintf("%s-%d", state.SC.Name, fileNo), StartTime: fmt.Sprintf("%s", state.DT.UTC()), EndTime: "", Center: state.Orbit.Origin.Name, Trajectory: &traj, Bodyframe: nil, Geometry: nil, Label: &label, TrajectoryPlot: &plot}
					}
					if conf.AsCSV {
						fAsCSV.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", state.DT.UTC()))
						fAsCSV = createAsCSVCSVFile(fmt.Sprintf("%s-%d", conf.Filename, fileNo), conf, state.DT)
					}
					fileNo++
					// Force writing this data point now instead of creating N new files.
					prevStatePtr = &state
					// Update the pointer of the first state for this trajectory.
					firstStatePtr = &state

					if conf.Cosmo {
						// Only write one point every short while.
						if state.DT.After(prevStatePtr.DT.Add(1 * time.Minute)) {
							// Change the color
							for i := 0; i < 3; i++ {
								color[i] -= 0.2
								if color[i] > 1 {
									color[i]--
								} else if color[i] < 0 {
									color[i]++
								}
							}
							asTxt := CgInterpolatedState{JD: julian.TimeToJD(state.DT), Position: state.Orbit.R(), Velocity: state.Orbit.V()}
							if _, err := f.WriteString("\n" + asTxt.ToText()); err != nil {
								panic(err)
							}
						}
					}

					if conf.AsCSV {
						a, e, i, Ω, ω, ν, _, _, _ := state.Orbit.Elements()
						deltaT := state.DT.Sub(firstStatePtr.DT)
						days := deltaT.Hours() / 24
						asTxt := fmt.Sprintf("%s,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f", state.DT.UTC().Format("2006-01-02 15:04:05"), a, e, Rad2deg180(i), Rad2deg180(Ω), Rad2deg180(ω), Rad2deg180(ν), firstStatePtr.SC.FuelMass, deltaT.Hours(), days)
						if _, err := fAsCSV.WriteString("\n" + asTxt); err != nil {
							panic(err)
						}
					}
					continue
				}
			}
			// Only write one datapoint per simulation minute.
			if prevStatePtr != nil && state.DT.Sub(prevStatePtr.DT) < StepSize {
				continue
			}
			prevStatePtr = &state
			if conf.Cosmo {
				asTxt := CgInterpolatedState{JD: julian.TimeToJD(state.DT), Position: state.Orbit.R(), Velocity: state.Orbit.V()}
				if _, err := f.WriteString("\n" + asTxt.ToText()); err != nil {
					panic(err)
				}
			}
			if conf.AsCSV {
				a, e, i, Ω, ω, ν, _, _, _ := state.Orbit.Elements()
				asTxt := fmt.Sprintf("%s,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f", state.DT.UTC().Format("2006-01-02 15:04:05"), a, e, Rad2deg(i), Rad2deg(Ω), Rad2deg(ω), Rad2deg(ν))
				if conf.CSVAppend != nil {
					asTxt += "," + conf.CSVAppend(state)
				}
				if _, err := fAsCSV.WriteString("\n" + asTxt); err != nil {
					panic(err)
				}
			}
		} else {
			// The channel is closed, hence the simulation is over.
			if conf.Cosmo {
				f.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", prevStatePtr.DT.UTC()))
				f.Close()
			}
			if conf.AsCSV {
				fAsCSV.WriteString(fmt.Sprintf("\n# Simulation time end (UTC): %s\n", prevStatePtr.DT.UTC()))
				fAsCSV.Close()
			}
			longerEnd := prevStatePtr.DT.Add(time.Duration(24) * time.Hour)
			if conf.Cosmo {
				curCgItem.EndTime = fmt.Sprintf("%s", longerEnd.UTC())
				curCgItem.TrajectoryPlot.Duration = fmt.Sprintf("%d d", int(longerEnd.Sub(firstStatePtr.DT).Hours()/24+1))
				cgItems = append(cgItems, curCgItem)
			}
			break
		}
	}
}

// ExportConfig configures the exporting of the simulation.
type ExportConfig struct {
	Filename     string
	Cosmo        bool
	AsCSV        bool
	Timestamp    bool
	CSVAppend    func(st MissionState) string // Custom export (do not include leading comma)
	CSVAppendHdr func() string                // Header for the custom export
}

// IsUseless returns whether this config doesn't actually do anything.
func (c ExportConfig) IsUseless() bool {
	return !c.Cosmo && !c.AsCSV
}
