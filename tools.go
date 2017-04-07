package smd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/gonum/floats"
	"github.com/gonum/matrix/mat64"
)

// TransferType defines the type of Lambert transfer
type TransferType uint8

// Longway returns whether or not this is the long way.
func (t TransferType) Longway() bool {
	switch t {
	case TType1:
		fallthrough
	case TType3:
		return false
	case TType2:
		fallthrough
	case TType4:
		return true
	default:
		panic(fmt.Errorf("cannot determine whether long or short way for %s", t))
	}
}

// Revs returns the number of revolutions given the type.
func (t TransferType) Revs() float64 {
	switch t {
	case TTypeAuto:
		fallthrough // auto-revs is limited to zero revolutions
	case TType1:
		fallthrough
	case TType2:
		return 0
	case TType3:
		fallthrough
	case TType4:
		return 1
	default:
		panic("unknown transfer type")
	}
}

func (t TransferType) String() string {
	switch t {
	case TTypeAuto:
		return "auto-revs"
	case TType1:
		return "type-1"
	case TType2:
		return "type-2"
	case TType3:
		return "type-3"
	case TType4:
		return "type-4"
	default:
		panic("unknown transfer type")
	}
}

const (
	// TTypeAuto lets the Lambert solver determine the type
	TTypeAuto TransferType = iota + 1
	// TType1 is transfer of type 1 (zero revolution, short way)
	TType1
	// TType2 is transfer of type 2 (zero revolution, long way)
	TType2
	// TType3 is transfer of type 3 (one revolutions, short way)
	TType3
	// TType4 is transfer of type 4 (one revolutions, long way)
	TType4
	lambertε         = 1e-4                   // General epsilon
	lambertTlambertε = 1e-4                   // Time epsilon
	lambertνlambertε = (5e-5 / 180) * math.Pi // 0.00005 degrees
)

// Hohmann computes an Hohmann transfer. It returns the departure and arrival velocities, and the time of flight.
// To get final computations:
// ΔvInit = vDepature - vI
// ΔvFinal = vArrival - vF
func Hohmann(rI, vI, rF, vF float64, body CelestialObject) (vDeparture, vArrival float64, tof time.Duration) {
	aTransfer := 0.5 * (rI + rF)
	vDeparture = math.Sqrt((2 * body.GM() / rI) - (body.GM() / aTransfer))
	vArrival = math.Sqrt((2 * body.GM() / rF) - (body.GM() / aTransfer))
	tof = time.Duration(math.Pi*math.Sqrt(math.Pow(aTransfer, 3)/body.GM())) * time.Second
	return
}

// Lambert solves the Lambert boundary problem:
// Given the initial and final radii and a central body, it returns the needed initial and final velocities
// along with φ which is the square of the difference in eccentric anomaly. Note that the direction of motion
// is computed directly in this function to simplify the generation of Pork chop plots.
func Lambert(Ri, Rf *mat64.Vector, Δt0 time.Duration, ttype TransferType, body CelestialObject) (Vi, Vf *mat64.Vector, φ float64, err error) {
	// Initialize return variables
	Vi = mat64.NewVector(3, nil)
	Vf = mat64.NewVector(3, nil)
	// Sanity checks
	Rir, _ := Ri.Dims()
	Rfr, _ := Rf.Dims()
	if Rir != Rfr || Rir != 3 {
		err = errors.New("initial and final radii must be 3x1 vectors")
		return
	}
	Δt0Sec := Δt0.Seconds()
	rI := mat64.Norm(Ri, 2)
	rF := mat64.Norm(Rf, 2)
	cosΔν := mat64.Dot(Ri, Rf) / (rI * rF)
	// Compute the direction of motion
	νI := math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
	νF := math.Atan2(Rf.At(1, 0), Rf.At(0, 0))
	dm := 1.0
	if ttype == TType2 {
		dm = -1.0
	} else if ttype == TTypeAuto {
		Δν := math.Atan2(Rf.At(1, 0), Rf.At(0, 0)) - math.Atan2(Ri.At(1, 0), Ri.At(0, 0))
		if Δν > 2*math.Pi {
			Δν -= 2 * math.Pi
		} else if Δν < 0 {
			Δν += 2 * math.Pi
		}
		if Δν > math.Pi {
			dm = -1.0
		} // We don't do the < math.Pi case because that's the initial value anyway.
	}

	A := dm * math.Sqrt(rI*rF*(1+cosΔν))
	if νF-νI < lambertνlambertε && floats.EqualWithinAbs(A, 0, lambertε) {
		err = errors.New("cannot compute trajectory: Δν ~=0 and A ~=0")
		return
	}

	φup := 4 * math.Pow(math.Pi, 2) * math.Pow(ttype.Revs()+1, 2)
	φlow := -4 * math.Pi

	if ttype.Revs() > 0 {
		// Generate a bunch of φ
		Δtmin := 4000 * 24 * 3600.0
		φBound := 0.0

		for φP := 15.; φP < φup; φP += 0.1 {
			c2 := (1 - math.Cos(math.Sqrt(φP))) / φP
			c3 := (math.Sqrt(φP) - math.Sin(math.Sqrt(φP))) / math.Sqrt(math.Pow(φP, 3))
			y := rI + rF + A*(φP*c3-1)/math.Sqrt(c2)
			χ := math.Sqrt(y / c2)
			Δt := (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.μ)
			if Δtmin > Δt {
				Δtmin = Δt
				φBound = φP
			}
		}

		// Determine whether we are going up or down bounds.
		if ttype == TType3 {
			φlow = φup
			φup = φBound
		} else if ttype == TType4 {
			φlow = φBound
		}
	}
	// Initial guesses for c2 and c3
	c2 := 1 / 2.
	c3 := 1 / 6.
	var Δt, y float64
	var iteration uint
	for math.Abs(Δt-Δt0Sec) > lambertTlambertε {
		if iteration > 10000 {
			err = errors.New("did not converge after 10000 iterations")
			return
		}
		iteration++
		y = rI + rF + A*(φ*c3-1)/math.Sqrt(c2)
		if A > 0 && y < 0 {
			tmpIt := 0
			for y < 0 {
				φ += 0.1
				y = rI + rF + A*(φ*c3-1)/math.Sqrt(c2)
				if tmpIt > 10000 {
					err = errors.New("did not converge after 10000 attempts to increase φ")
					return
				}
				tmpIt++
			}
		}
		χ := math.Sqrt(y / c2)
		Δt = (math.Pow(χ, 3)*c3 + A*math.Sqrt(y)) / math.Sqrt(body.μ)
		if ttype != TType3 {
			if Δt <= Δt0Sec {
				φlow = φ
			} else {
				φup = φ
			}
		} else {
			if Δt >= Δt0Sec {
				φlow = φ
			} else {
				φup = φ
			}
		}
		φ = (φup + φlow) / 2
		if φ > lambertε {
			sφ := math.Sqrt(φ)
			ssφ, csφ := math.Sincos(sφ)
			c2 = (1 - csφ) / φ
			c3 = (sφ - ssφ) / math.Sqrt(math.Pow(φ, 3))
		} else if φ < -lambertε {
			sφ := math.Sqrt(-φ)
			c2 = (1 - math.Cosh(sφ)) / φ
			c3 = (math.Sinh(sφ) - sφ) / math.Sqrt(math.Pow(-φ, 3))
		} else {
			c2 = 1 / 2.
			c3 = 1 / 6.
		}
	}
	f := 1 - y/rI
	gDot := 1 - y/rF
	g := (A * math.Sqrt(y/body.μ))
	// Compute velocities
	Rf2 := mat64.NewVector(3, nil)
	Vi.AddScaledVec(Rf, -f, Ri)
	Vi.ScaleVec(1/g, Vi)
	Rf2.ScaleVec(gDot, Rf)
	Vf.AddScaledVec(Rf2, -1, Ri)
	Vf.ScaleVec(1/g, Vf)
	return
}

// PCPGenerator generates the PCP files to perform contour plots in Matlab (and eventually prints the command).
func PCPGenerator(initPlanet, arrivalPlanet CelestialObject, initLaunch, maxLaunch, initArrival, maxArrival time.Time, ptsPerLaunchDay, ptsPerArrivalDay float64, plotC3 bool, pcpName string, verbose bool) (c3Map, tofMap, vinfMap map[time.Time][]float64, vInfInitVecs, vInfArriVecs map[time.Time][]mat64.Vector) {
	launchWindow := int(maxLaunch.Sub(initLaunch).Hours() / 24)    //days
	arrivalWindow := int(maxArrival.Sub(initArrival).Hours() / 24) //days
	// Create the output arrays
	c3Map = make(map[time.Time][]float64)
	tofMap = make(map[time.Time][]float64)
	vinfMap = make(map[time.Time][]float64)
	vInfInitVecs = make(map[time.Time][]mat64.Vector)
	vInfArriVecs = make(map[time.Time][]mat64.Vector)
	if verbose {
		fmt.Printf("Launch window: %d days\nArrival window: %d days\n", launchWindow, arrivalWindow)
	}
	// Stores the content of the dat file.
	// No trailing new line because it's add in the for loop.
	dat := fmt.Sprintf("%% %s -> %s\n%%arrival days as new lines, departure as new columns", initPlanet, arrivalPlanet)
	hdls := make([]*os.File, 4)
	var fNames []string
	if plotC3 {
		fNames = []string{"c3", "tof", "vinf", "dates"}
	} else {
		fNames = []string{"vinf-init", "tof", "vinf-arrival", "dates"}
	}
	for i, name := range fNames {
		// Write CSV file.
		f, err := os.Create(fmt.Sprintf("./contour-%s-%s.dat", pcpName, name))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(dat); err != nil {
			panic(err)
		}
		hdls[i] = f
	}

	// Let's write the date information now and close that file.
	hdls[3].WriteString(fmt.Sprintf("\n%%departure: \"%s\"\n%%arrival: \"%s\"\n%d,%d\n%d,%d\n", initLaunch.Format("2006-Jan-02"), initArrival.Format("2006-Jan-02"), 1, launchWindow, 1, arrivalWindow))
	hdls[3].Close()

	for launchDay := 0.; launchDay < float64(launchWindow); launchDay += 1 / ptsPerLaunchDay {
		// New line in files
		for _, hdl := range hdls[:3] {
			if _, err := hdl.WriteString("\n"); err != nil {
				panic(err)
			}
		}
		launchDT := initLaunch.Add(time.Duration(launchDay*24*3600) * time.Second)
		if verbose {
			fmt.Printf("Launch date %s\n", launchDT)
		}
		// Initialize the values
		c3Map[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		tofMap[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		vinfMap[launchDT] = make([]float64, arrivalWindow*int(ptsPerArrivalDay))
		vInfInitVecs[launchDT] = make([]mat64.Vector, arrivalWindow*int(ptsPerArrivalDay))
		vInfArriVecs[launchDT] = make([]mat64.Vector, arrivalWindow*int(ptsPerArrivalDay))

		initOrbit := initPlanet.HelioOrbit(launchDT)
		initPlanetR := mat64.NewVector(3, initOrbit.R())
		initPlanetV := mat64.NewVector(3, initOrbit.V())
		arrivalIdx := 0
		for arrivalDay := 0.; arrivalDay < float64(arrivalWindow); arrivalDay += 1 / ptsPerArrivalDay {
			arrivalDT := initArrival.Add(time.Duration(arrivalDay*24) * time.Hour)
			arrivalOrbit := arrivalPlanet.HelioOrbit(arrivalDT)
			arrivalR := mat64.NewVector(3, arrivalOrbit.R())
			arrivalV := mat64.NewVector(3, arrivalOrbit.V())

			tof := arrivalDT.Sub(launchDT)
			Vi, Vf, _, err := Lambert(initPlanetR, arrivalR, tof, TTypeAuto, Sun)
			var c3, vInfArrival float64
			if err != nil {
				if verbose {
					fmt.Printf("departure: %s\tarrival: %s\t\t%s\n", launchDT, arrivalDT, err)
				}
				c3 = math.NaN()
				vInfArrival = math.NaN()
				// Store a nil vector to not loose track of indexing
				vInfInitVecs[launchDT][arrivalIdx] = *mat64.NewVector(3, nil)
				vInfArriVecs[launchDT][arrivalIdx] = *mat64.NewVector(3, nil)
			} else {
				// Compute the c3
				VInfInit := mat64.NewVector(3, nil)
				VInfInit.SubVec(initPlanetV, Vi)
				// WARNING: When *not* plotting the c3, we just store the V infinity at departure in the c3 variable!
				if plotC3 {
					c3 = math.Pow(mat64.Norm(VInfInit, 2), 2)
				} else {
					c3 = mat64.Norm(VInfInit, 2)
				}
				if math.IsNaN(c3) {
					c3 = 0
				}
				// Compute the v_infinity at destination
				VInfArrival := mat64.NewVector(3, nil)
				VInfArrival.SubVec(arrivalV, Vf)
				vInfArrival = mat64.Norm(VInfArrival, 2)
				vInfInitVecs[launchDT][arrivalIdx] = *VInfInit
				vInfArriVecs[launchDT][arrivalIdx] = *VInfArrival
			}
			// Store data in the files
			hdls[0].WriteString(fmt.Sprintf("%f,", c3))
			hdls[1].WriteString(fmt.Sprintf("%f,", tof.Hours()/24))
			hdls[2].WriteString(fmt.Sprintf("%f,", vInfArrival))
			// and in the arrays
			c3Map[launchDT][arrivalIdx] = c3
			tofMap[launchDT][arrivalIdx] = tof.Hours() / 24
			vinfMap[launchDT][arrivalIdx] = vInfArrival
			arrivalIdx++
		}
	}
	if verbose {
		// Print the matlab command to help out
		if plotC3 {
			fmt.Printf("=== MatLab ===\npcpplots('%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), arrivalPlanet.Name)
		} else {
			fmt.Printf("=== MatLab ===\npcpplotsVinfs('%s', '%s', '%s', '%s', '%s')\n", pcpName, initLaunch.Format("2006-01-02"), initArrival.Format("2006-01-02"), initPlanet.Name, arrivalPlanet.Name)
		}
	}
	return
}
