package smd

import (
	"testing"
	"time"
)

func TestEstimate(t *testing.T) {
	// Test that an estimate does propagate the same way as "Mission".
	perts := Perturbations{Jn: 2}
	startDT := time.Now().UTC()
	duration := time.Duration(24) * time.Hour
	endDT := startDT.Add(duration)
	// Define the orbits
	leoMission := NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, Earth)
	leoEstimate := *NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, Earth)
	// Initialize the mission and estimates
	mission := NewPreciseMission(NewEmptySC("LEO", 0), leoMission, startDT, endDT, Cartesian, perts, time.Second, ExportConfig{})
	orbitEstimate := NewOrbitEstimate("estimator", leoEstimate, perts, startDT, time.Second)
	// Run
	mission.Propagate()
	orbitEstimate.PropagateUntil(endDT)
	finalEst := orbitEstimate.State()
	finalR, finalV := finalEst.Orbit.RV()
	missionR, missionV := leoMission.RV()
	if !finalEst.DT.UTC().Equal(mission.CurrentDT.UTC()) {
		t.Logf("\nEst.:%s\nMis.:%s", finalEst.DT.UTC(), mission.CurrentDT.UTC())
		t.Fatal("incorrect ending date")
	}
	if !vectorsEqual(finalR, missionR) || !vectorsEqual(finalV, missionV) {
		t.Logf("\nEst.: R=%+v\tV=%+v\nMis.: R=%+v\tV=%+v\t(truth)", finalR, finalV, missionR, missionV)
		t.Fatal("incorrect final vectors")
	}
}
