package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ChristopherRabotin/smd"
	"github.com/gonum/matrix/mat64"
)

func main() {
	/*** CONFIG ***/
	initPlanet := smd.Earth
	initLaunch := time.Date(2005, 6, 1, 0, 0, 0, 0, time.UTC)
	arrivalPlanet := smd.Mars
	initArrival := time.Date(2005, 12, 1, 0, 0, 0, 0, time.UTC)
	launchWindow := 500  //days
	arrivalWindow := 500 //days
	/*** END CONFIG ***/

	if launchWindow != arrivalWindow {
		panic("launch and arrival window must be of same length for now")
	}

	c3Vec := make([]float64, launchWindow+arrivalWindow)
	vInfVec := make([]float64, launchWindow+arrivalWindow)
	tofVec := make([]float64, launchWindow+arrivalWindow)

	for launchDay := 0; launchDay < launchWindow; launchDay++ {
		launchDT := initLaunch.Add(time.Duration(launchDay*24) * time.Hour)
		fmt.Printf("Launch date %s\n", launchDT)
		initOrbit := initPlanet.HelioOrbit(launchDT)
		initR := mat64.NewVector(3, initOrbit.R())
		initV := mat64.NewVector(3, initOrbit.V())
		for arrivalDay := 0; arrivalDay < arrivalWindow; arrivalDay++ {
			arrivalDT := initArrival.Add(time.Duration(arrivalDay*24) * time.Hour)
			arrivalOrbit := arrivalPlanet.HelioOrbit(arrivalDT)
			arrivalR := mat64.NewVector(3, arrivalOrbit.R())

			tof := arrivalDT.Sub(launchDT)
			tofVec[launchDay+arrivalDay] = tof.Hours() / 24
			Vi, Vf, _, err := smd.Lambert(initR, arrivalR, tof, smd.TType2, smd.Sun)
			if err != nil {
				fmt.Println(err)
				// Zero padding
				c3Vec[launchDay+arrivalDay] = 0
				vInfVec[launchDay+arrivalDay] = 0
				break
			}
			// Compute the c3
			VInfInit := mat64.NewVector(3, nil)
			VInfInit.SubVec(Vi, initV)
			c3 := math.Pow(mat64.Norm(VInfInit, 2), 2)
			if math.IsNaN(c3) {
				c3 = 0
			}
			c3Vec[launchDay+arrivalDay] = c3
			// Compute the v_infinity at destination
			VInfArrival := mat64.NewVector(3, nil)
			VInfArrival.SubVec(VInfArrival, Vf)
			vInfArrival := mat64.Norm(VInfArrival, 2)
			vInfVec[launchDay+arrivalDay] = vInfArrival
		}
	}

	// Create matrix, which is fully sequential (i.e. one column).
	// Vectors to create the .mat file, cf. https://kst-plot.kde.org/kst1/handbook/data-types.html#extract-mx .
	for _, data := range []struct {
		name string
		vec  []float64
	}{{"c3", c3Vec}, {"tof", tofVec}, {"vInf", vInfVec}} {
		matContent := fmt.Sprintf("# %s -> %s\n", initPlanet, arrivalPlanet)
		matContent += fmt.Sprintf("[MATRIX,%d,0,0,1,1]\n", launchWindow)
		for launchDay := 0; launchDay < launchWindow; launchDay++ {
			matContent += fmt.Sprintf("%d\n", launchDay)
		}
		for arrivalDay := 0; arrivalDay < arrivalWindow; arrivalDay++ {
			matContent += fmt.Sprintf("%d\n", arrivalDay)
		}

		for i := 0; i < len(data.vec); i++ {
			matContent += fmt.Sprintf("%f\n", data.vec[i])
		}

		// Write CSV file.
		f, err := os.Create(fmt.Sprintf("./data-%s.dat", data.name))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(matContent); err != nil {
			panic(err)
		}
	}

}
