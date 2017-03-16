package main

import (
	"fmt"
	"math"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
	"github.com/soniakeys/meeus/julian"
)

const (
	part     = 1
	question = 1
)

func main() {
	launchDT := julian.JDToTime(2453755.29167)
	jgaDT := julian.JDToTime(2454159.73681)
	pceDT := julian.JDToTime(2457217.99931)
	fmt.Printf("Launch: %s\nJGA: %s\nPluto enc.: %s\n\n", launchDT, jgaDT, pceDT)

	fmt.Println("==   PART 1   ==")
	fmt.Println("== QUESTION 1 ==")
	// Get the positions of Earth and Jupiter
	earthAtLaunch := smd.Earth.HelioOrbit(launchDT)
	jupiterAtJGA := smd.Jupiter.HelioOrbit(jgaDT)
	earthR, earthV := earthAtLaunch.RV()
	jupiterR, jupiterV := jupiterAtJGA.RV()
	earthRVec := mat64.NewVector(3, nil)
	earthVVec := mat64.NewVector(3, nil)
	jupiterRVec := mat64.NewVector(3, nil)
	jupiterVVec := mat64.NewVector(3, nil)
	for i := 0; i < 3; i++ {
		earthRVec.SetVec(i, earthR[i])
		earthVVec.SetVec(i, earthV[i])
		jupiterRVec.SetVec(i, jupiterR[i])
		jupiterVVec.SetVec(i, jupiterV[i])
	}
	ViLaunch, VfJGA, _, err := smd.Lambert(earthRVec, jupiterRVec, jgaDT.Sub(launchDT), smd.TTypeAuto, smd.Sun)
	// Compute the V_inifities
	if err != nil {
		panic(fmt.Errorf("error while solving Lambert: %s", err))
	}
	// Compute the c3
	VInfInit := mat64.NewVector(3, nil)
	VInfInit.SubVec(ViLaunch, earthVVec)
	c3 := math.Pow(mat64.Norm(VInfInit, 2), 2)
	if math.IsNaN(c3) {
		c3 = 0
	}
	// Compute the DLA and RLA at launch
	rla := smd.Rad2deg(math.Atan2(VInfInit.At(1, 0), VInfInit.At(0, 0)))
	dla := smd.Rad2deg(math.Atan2(VInfInit.At(2, 0), mat64.Norm(VInfInit, 2)))
	// Compute the v_infinity at destination
	VInfInJGA := mat64.NewVector(3, nil)
	VInfInJGA.SubVec(jupiterVVec, VfJGA)
	vInfInJGA := mat64.Norm(VInfInJGA, 2)
	fmt.Printf("c3: %f km^2/s^2\tRLA: %f deg\tDLA: %f deg\nV_inf@JGA: %f km/s\n", c3, rla, dla, vInfInJGA)

	fmt.Println("== QUESTION 2 ==")
	// Get the positions of Earth and Jupiter
	plutoAtPCE := smd.Pluto.HelioOrbit(pceDT)
	plutoR, plutoV := plutoAtPCE.RV()
	plutoRVec := mat64.NewVector(3, nil)
	plutoVVec := mat64.NewVector(3, nil)
	for i := 0; i < 3; i++ {
		plutoRVec.SetVec(i, plutoR[i])
		plutoVVec.SetVec(i, plutoV[i])
	}
	ViJupiter, VfPCE, _, err := smd.Lambert(jupiterRVec, plutoRVec, pceDT.Sub(jgaDT), smd.TTypeAuto, smd.Sun)
	// Compute the V_inifities
	if err != nil {
		panic(fmt.Errorf("error while solving Lambert: %s", err))
	}
	// Compute the v inf out
	VInfOutJGA := mat64.NewVector(3, nil)
	VInfOutJGA.SubVec(jupiterVVec, ViJupiter)
	// Compute the v_infinity at destination
	VInfInPCE := mat64.NewVector(3, nil)
	VInfInPCE.SubVec(VfPCE, plutoVVec)
	vInfInPCE := mat64.Norm(VInfInPCE, 2)
	fmt.Printf("V_inf,out@JGA: %f km/s\nV_inf,in@PCE: %f km/s\n", mat64.Norm(VInfOutJGA, 2), vInfInPCE)

	fmt.Println("== QUESTION 3 ==\nManual")
	fmt.Println("== QUESTION 4 ==")
	ψ, rP, bT, bR, _, θ := smd.GAFromVinf([]float64{VInfInJGA.At(0, 0), VInfInJGA.At(1, 0), VInfInJGA.At(2, 0)}, []float64{VInfOutJGA.At(0, 0), VInfOutJGA.At(1, 0), VInfOutJGA.At(2, 0)}, smd.Jupiter)
	fmt.Printf("rP = %f km (hP = %f km)\tBT = %f km\tBR = %f km\tψ = %f\tθ = %f\n", rP, rP-smd.Jupiter.Radius, bT, bR, smd.Rad2deg(ψ), smd.Rad2deg(θ))

	fmt.Println("== QUESTION 5 ==")
	ΔVhelio := mat64.NewVector(3, nil)
	ΔVhelio.SubVec(ViJupiter, VfJGA)
	fmt.Printf("ΔVhelio = %+v km/s\t|ΔVhelio| = %f km/s\n", mat64.Formatted(ΔVhelio.T()), mat64.Norm(ΔVhelio, 2))

	fmt.Println("==   PART 2   ==")
	// The rest is done via the pcpplot command
	fmt.Printf("=== TOF ===\nEarth -> Jupiter: %.2f days\nJupiter -> Pluto: %.2f days\n", jgaDT.Sub(launchDT).Hours()/24, pceDT.Sub(jgaDT).Hours()/24)
	// Find the point to plot
	pcp2InitLaunch := julian.JDToTime(2454129.5)
	pcp2InitArrival := julian.JDToTime(2456917.5)
	fmt.Printf("hold on; plot(%.3f, %.3f, 'g*', 'MarkerSize', 20)\n", jgaDT.Sub(pcp2InitLaunch).Hours()/24, pceDT.Sub(pcp2InitArrival).Hours()/24)
}
