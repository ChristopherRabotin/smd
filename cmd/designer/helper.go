package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/soniakeys/meeus/julian"
)

// Result stores a full valid result.
type Result struct {
	launch       time.Time
	c3, rla, dla float64
	flybys       []GAResult
	arrival      time.Time
	vInf         float64
}

// CSV returns the CSV of this result
func (r Result) CSV() string {
	rtn := fmt.Sprintf("%.3f (%s),%f,%f,%f,", julian.TimeToJD(r.launch), r.launch.Format(dateFormat), r.c3, r.rla, r.dla)
	for _, flyby := range r.flybys {
		rtn += flyby.CSV()
	}
	rtn += fmt.Sprintf("%.3f (%s),%f", julian.TimeToJD(r.arrival), r.arrival.Format(dateFormat), r.vInf)
	return rtn
}

//Clone figure it out
func (r Result) Clone() Result {
	newResult := Result{r.launch, r.c3, r.rla, r.dla, nil, r.arrival, r.vInf}
	newResult.flybys = make([]GAResult, len(r.flybys))
	for i, fb := range r.flybys {
		newResult.flybys[i] = fb
	}
	return newResult
}

// NewResult initializes a result.
func NewResult(launch time.Time, c3, rla, dla float64, numGA int) Result {
	return Result{launch, c3, rla, dla, make([]GAResult, numGA), time.Now(), -1}
}

// GAResult stores the result of a gravity assist.
type GAResult struct {
	DT     time.Time
	deltaV float64
	radius float64
	bt, br float64
	phi    float64 // Only used in the case of a resonant orbit
}

// CSV returns the CSV of this result
func (g GAResult) CSV() string {
	if g.DT != (time.Time{}) {
		if g.phi != 0 {
			return fmt.Sprintf("%.3f (%s),%f,%f,%f,%f,%f,", julian.TimeToJD(g.DT), g.DT.Format(dateFormat), g.deltaV, g.radius, g.bt, g.br, smd.Rad2deg(g.phi))
		}
		return fmt.Sprintf("%.3f (%s),%f,%f,%f,%f,", julian.TimeToJD(g.DT), g.DT.Format(dateFormat), g.deltaV, g.radius, g.bt, g.br)
	}
	return ""
}

// StreamResults is used to stream the results to the output file.
func StreamResults(prefix string, planets []smd.CelestialObject, rsltChan <-chan (Result)) {
	f, err := os.Create(fmt.Sprintf("%s/%s-results.csv", outputdir, prefix))
	if err != nil {
		panic(err)
	}
	hdrs := "launch,c3,rla,dla,"
	for pNo, planet := range planets[0:len(planets)] {
		hdrs += planet.Name + "DT,"
		hdrs += planet.Name + "DV,"
		hdrs += planet.Name + "Rp,"
		hdrs += planet.Name + "Bt,"
		hdrs += planet.Name + "Br,"
		if pNo > 0 && pNo < len(flybys) && flybys[pNo].isResonant {
			// Repeat
			hdrs += planet.Name + "-ResDT,"
			hdrs += planet.Name + "-ResDV,"
			hdrs += planet.Name + "-ResRp,"
			hdrs += planet.Name + "-ResBt,"
			hdrs += planet.Name + "-ResBr,"
			hdrs += planet.Name + "-ResPhi,"
		}
	}
	hdrs += "arrival,vInf,Bt,Br\n"
	if _, err := f.WriteString(hdrs); err != nil {
		panic(err)
	}
	for rslt := range rsltChan {
		f.WriteString(rslt.CSV() + "\n")
		wg.Done()
	}
	wg.Done() // Done writing everything.
}

type target struct {
	BT1, BT2, BR1, BR2, Assocψ, Rp1, Rp2 float64
	ega1Vin, ega1Vout, ega2Vin, ega2Vout float64
}

func (t target) String() string {
	return fmt.Sprintf("ψ=%f ===\nGA1: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n\nGA2: Bt=%f\tBr=%f\trP=%f\nVin=%f\tVout=%f\tdelta=%f\n", smd.Rad2deg(t.Assocψ), t.BT1, t.BR1, t.Rp1, t.ega1Vin, t.ega1Vout, t.ega1Vout-t.ega1Vin, t.BT2, t.BR2, t.Rp2, t.ega2Vin, t.ega2Vout, t.ega2Vout-t.ega2Vin)
}
