package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
)

// Result stores a full valid result.
type Result struct {
	launch  time.Time
	c3      float64
	flybys  []GAResult
	arrival time.Time
	vInf    float64
}

// CSV returns the CSV of this result
func (r Result) CSV() string {
	rtn := fmt.Sprintf("%s,%f,", r.launch, r.c3)
	for _, flyby := range r.flybys {
		rtn += flyby.CSV()
	}
	rtn += fmt.Sprintf("%s,%f,", r.arrival, r.vInf)
	return rtn
}

//Clone figure it out
func (r Result) Clone() Result {
	newResult := Result{r.launch, r.c3, nil, r.arrival, r.vInf}
	newResult.flybys = make([]GAResult, len(r.flybys))
	for i, fb := range r.flybys {
		newResult.flybys[i] = fb
	}
	return newResult
}

// NewResult initializes a result.
func NewResult(launch time.Time, c3 float64, numGA int) Result {
	return Result{launch, c3, make([]GAResult, numGA), time.Now(), -1}
}

// GAResult stores the result of a gravity assist.
type GAResult struct {
	DT     time.Time
	deltaV float64
	radius float64
}

// CSV returns the CSV of this result
func (g GAResult) CSV() string {
	if g.DT != (time.Time{}) {
		return fmt.Sprintf("%s,%f,%f,", g.DT, g.deltaV, g.radius)
	}
	return ""
}

// StreamResults is used to stream the results to the output file.
func StreamResults(prefix string, planets []smd.CelestialObject, rsltChan <-chan (Result)) {
	f, err := os.Create(fmt.Sprintf("./%s-results.csv", prefix))
	if err != nil {
		panic(err)
	}
	hdrs := "launch,c3,"
	for _, planet := range planets[0 : len(planets)-1] {
		hdrs += planet.Name + "DT,"
		hdrs += planet.Name + "DV,"
		hdrs += planet.Name + "Rp,"
	}
	hdrs += "arrival,vInf\n"
	if _, err := f.WriteString(hdrs); err != nil {
		panic(err)
	}
	for rslt := range rsltChan {
		f.WriteString(rslt.CSV() + "\n")
	}
}
