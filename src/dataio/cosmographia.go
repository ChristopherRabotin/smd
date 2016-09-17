package dataio

import (
	"errors"
	"fmt"
	"strings"
)

// CgCatalog definiton.
type CgCatalog struct {
	Version string     `json:"version"`
	Name    string     `json:"name"`
	Items   []*CgItems `json:"items"`
	Require []*string  `json:"require"`
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
	Trajectory     *CgTrajectory     `json:"trajectory"`
	Bodyframe      *CgBodyFrame      `json:"bodyFrame"`
	Geometry       *CgGeometry       `json:"geometry"`
	Label          *CgLabel          `json:"label"`
	TrajectoryPlot *CgTrajectoryPlot `json:"trajectoryPlot"`
}

// CgTrajectory definition.
type CgTrajectory struct {
	Type   string `json:"type"`
	Source string `json:"source"`
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
	Type string `json:"type"`
	Name string `json:"name"`
}

func (c *CgBodyFrame) String() string {
	return c.Name + " (type: " + c.Type + ")"
}

// CgGeometry definiton.
type CgGeometry struct {
	Type   string    `json:"type"`
	Mesh   []float64 `json:"meshRotation"`
	Size   float64   `json:"size"`
	Source string    `json:"source"`
}

// CgLabel definiton.
type CgLabel struct {
	Color    []float64 `json:"color"`
	FadeSize int       `json:"fadeSize"`
	ShowText bool      `json:"showText"`
}

func (l *CgLabel) String() string {
	return fmt.Sprintf("color %v, fade %d, show %v", l.Color, l.FadeSize, l.ShowText)
}

// CgTrajectoryPlot definition.
type CgTrajectoryPlot struct {
	Color       []float64 `json:"color"`
	LineWidth   int       `json:"lineWidth"`
	Duration    string    `json:"duration"`
	Lead        string    `json:"lead"`
	Fade        int       `json:"fade"`
	SampleCount int       `json:"sampleCount"`
}
