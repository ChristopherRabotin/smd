package smd

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonum/matrix/mat64"
)

func TestEstimate(t *testing.T) {
	// Test that an estimate does propagate the same way as "Mission".
	perts := Perturbations{Jn: 3}
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
	// Test Φ
	var inv, id mat64.Dense
	if ierr := inv.Inverse(orbitEstimate.Φ); ierr != nil {
		t.Fatalf("could not inverse Φ: %s ", ierr)
	}
	id.Mul(orbitEstimate.Φ, &inv)
	t.Logf("\n%+v", mat64.Formatted(&id))
}

func TestEstimatePhi(t *testing.T) {
	perts := Perturbations{Jn: 3}
	startDT := time.Now().UTC()
	duration0 := time.Duration(30) * time.Second
	duration2 := time.Duration(15) * time.Second
	endDT := startDT.Add(duration0)
	endDT1 := startDT.Add(duration2)
	leoEstimate := *NewOrbitFromOE(7000, 0.00001, 30, 80, 40, 0, Earth)
	est0 := NewOrbitEstimate("full", leoEstimate, perts, startDT, time.Second)
	est0.PropagateUntil(endDT)
	est1 := NewOrbitEstimate("part1", leoEstimate, perts, startDT, time.Second)
	est1.PropagateUntil(endDT1)
	// Start the second estimate from the end of the first one.
	state := est1.State()
	est2 := NewOrbitEstimate("part2", state.Orbit, perts, startDT.Add(duration2), time.Second)
	est2.PropagateUntil(endDT)
	var est1ΦInv mat64.Dense
	if ierr := est1ΦInv.Inverse(est1.Φ); ierr != nil {
		panic(fmt.Errorf("could not invert `est1.Φ`: %s", ierr))
	}
	var Φ2 mat64.Dense
	Φ2.Mul(est0.Φ, &est1ΦInv)
	if !mat64.EqualApprox(&Φ2, est2.Φ, 1e-12) {
		t.Logf("\n%+v", mat64.Formatted(&Φ2))
		t.Fatal("did not get Φ2 correctly")
	}

}
