package smd

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ChristopherRabotin/ode"
	"github.com/gonum/matrix/mat64"
)

const (
	// StepSize is the default step size of propagation.
	StepSize = 10 * time.Second
)

var wg sync.WaitGroup

/* Handles the astrodynamical propagations. */

// Mission defines a mission and does the propagation.
type Mission struct {
	Vehicle                    *Spacecraft  // As pointer because SC may be altered during propagation.
	Orbit                      *Orbit       // As pointer because the orbit changes during propagation.
	Φ                          *mat64.Dense // STM
	StartDT, StopDT, CurrentDT time.Time
	perts                      Perturbations
	step                       time.Duration // time step
	stopChan                   chan (bool)
	histChans                  []chan (State)
	computeSTM, done, collided bool
	autoChanClosing            bool // Set to False to not automatically close the channels upon end propgation time reached.
}

// NewMission is the same as NewPreciseMission with the default step size.
func NewMission(s *Spacecraft, o *Orbit, start, end time.Time, perts Perturbations, computeSTM bool, conf ExportConfig) *Mission {
	return NewPreciseMission(s, o, start, end, perts, StepSize, computeSTM, conf)
}

// NewPreciseMission returns a new Mission instance with custom provided time step.
func NewPreciseMission(s *Spacecraft, o *Orbit, start, end time.Time, perts Perturbations, step time.Duration, computeSTM bool, conf ExportConfig) *Mission {
	// Must switch to UTC as all ephemeris data is in UTC.
	if start.Location() != time.UTC {
		start = start.UTC()
	}
	if end.Location() != time.UTC {
		end = end.UTC()
	}
	rSTM, _ := perts.STMSize()
	a := &Mission{s, o, DenseIdentity(rSTM), start, end, start, perts, step, make(chan (bool), 1), nil, computeSTM, false, false, true}
	// Create a main history channel if there is any exporting
	if !conf.IsUseless() {
		a.histChans = []chan (State){make(chan (State), 10)}
		wg.Add(1)
		go func() {
			defer wg.Done()
			StreamStates(conf, a.histChans[0])
		}()
		// Write the first data point.
		a.histChans[0] <- State{a.CurrentDT, *s, *o, nil, nil}
	}

	if end.Before(start) {
		a.Vehicle.logger.Log("level", "warning", "subsys", "astro", "message", "no end date")
	}

	return a
}

// RegisterStateChan appends a new channel where to publish states as they are computed
// WARNING: One *should not* write to this channel, but no check is done. Don't be dumb.
func (a *Mission) RegisterStateChan(c chan (State)) {
	a.histChans = append(a.histChans, c)
}

// LogStatus returns the status of the propagation and vehicle.
func (a *Mission) LogStatus() {
	a.Vehicle.logger.Log("level", "info", "subsys", "astro", "date", a.CurrentDT, "fuel(kg)", a.Vehicle.FuelMass, "orbit", a.Orbit)
}

// PropagateUntil propagates until the given time is reached.
func (a *Mission) PropagateUntil(dt time.Time, autoClose bool) {
	a.autoChanClosing = autoClose
	a.StopDT = dt
	// For the final propagation report says the exact prop time, we update the start date time.
	a.StartDT = a.CurrentDT
	a.Propagate()
}

// Propagate starts the propagation.
func (a *Mission) Propagate() {
	// Add a ticker status report based on the duration of the simulation.
	a.LogStatus()
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for _ = range ticker.C {
			if a.done {
				break
			}
			a.LogStatus()
		}
	}()
	vInit := Norm(a.Orbit.V())
	initFuel := a.Vehicle.FuelMass
	ode.NewRK4(0, a.step.Seconds(), a).Solve() // Blocking.
	vFinal := Norm(a.Orbit.V())
	a.done = true
	duration := a.CurrentDT.Sub(a.StartDT)
	durStr := duration.String()
	if duration.Hours() > 24 {
		durStr += fmt.Sprintf(" (~%.3fd)", duration.Hours()/24)
	}
	a.Vehicle.logger.Log("level", "notice", "subsys", "astro", "status", "finished", "duration", durStr, "Δv(km/s)", math.Abs(vFinal-vInit), "fuel(kg)", initFuel-a.Vehicle.FuelMass)
	a.LogStatus()
	if a.Vehicle.handleFuel && a.Vehicle.FuelMass < 0 {
		a.Vehicle.logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", a.Vehicle.FuelMass)
		panic("cannot continue without fuel")
	}
	wg.Wait() // Don't return until we're done writing all the files.
}

// StopPropagation is used to stop the propagation before it is completed.
func (a *Mission) StopPropagation() {
	a.stopChan <- true
}

// Stop implements the stop call of the integrator. To stop the propagation, call StopPropagation().
func (a *Mission) Stop(t float64) bool {
	select {
	case <-a.stopChan:
		for _, histChan := range a.histChans {
			close(histChan)
		}
		return true // Stop because there is a request to stop.
	default:
		a.CurrentDT = a.CurrentDT.Add(a.step) // XXX: Should this be in SetState?
		if a.StopDT.Before(a.StartDT) {
			// A hard limit is set on a ten year propagation.
			kill := false
			if a.CurrentDT.After(a.StartDT.Add(24 * 3652.5 * time.Hour)) {
				a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "status", "killed")
				kill = true
			}
			if !kill {
				// Check if any waypoint still needs to be reached.
				for _, wp := range a.Vehicle.WayPoints {
					if !wp.Cleared() {
						return false
					}
				}
			}
			for _, histChan := range a.histChans {
				close(histChan)
			}
			return true
		}
		if a.CurrentDT.Sub(a.StopDT).Nanoseconds() > 0 {
			if a.autoChanClosing {
				for _, histChan := range a.histChans {
					close(histChan)
				}
			}
			return true // Stop, we've reached the end of the simulation.
		}
	}
	return false
}

// GetState returns the state for the integrator for the Gaussian VOP.
func (a *Mission) GetState() (s []float64) {
	stateSize := 7
	if a.computeSTM {
		rSTM, cSTM := a.perts.STMSize()
		stateSize += rSTM * cSTM
		if a.perts.Drag {
			stateSize += 1
		}
	}
	s = make([]float64, stateSize)
	R, V := a.Orbit.RV()
	// R, V in the state
	for i := 0; i < 3; i++ {
		s[i] = R[i]
		s[i+3] = V[i]
	}
	s[6] = a.Vehicle.FuelMass
	if a.computeSTM {
		// Add the components of Φ
		rSTM, cSTM := a.perts.STMSize()
		sIdx := rSTM + 1
		for i := 0; i < rSTM; i++ {
			for j := 0; j < cSTM; j++ {
				s[sIdx] = a.Φ.At(i, j)
				sIdx++
			}
		}
	}
	return
}

// SetState sets the updated state.
func (a *Mission) SetState(t float64, s []float64) {
	R := []float64{s[0], s[1], s[2]}
	V := []float64{s[3], s[4], s[5]}
	*a.Orbit = *NewOrbitFromRV(R, V, a.Orbit.Origin) // Deref is important (cd. TestMissionSpiral)

	// Orbit sanity checks and warnings.
	if !a.collided && a.Orbit.RNorm() < a.Orbit.Origin.Radius {
		a.collided = true
		a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "collided", a.Orbit.Origin.Name, "dt", a.CurrentDT, "r", a.Orbit.RNorm(), "radius", a.Orbit.Origin.Radius)
	} else if a.collided && a.Orbit.RNorm() > a.Orbit.Origin.Radius*1.1 {
		// Now further from the 10% dead zone
		a.collided = false
		a.Vehicle.logger.Log("level", "critical", "subsys", "astro", "revived", a.Orbit.Origin.Name, "dt", a.CurrentDT)
	}

	// Propulsion sanity check
	if a.Vehicle.handleFuel && a.Vehicle.FuelMass < 0 && s[6] <= 0 {
		a.Vehicle.logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", s[6])
		panic("cannot continue without fuel")
	}
	a.Vehicle.FuelMass = s[6]

	var latestVector *mat64.Vector
	if a.Vehicle.Drag > 0 && a.computeSTM {
		st := s[0:6]
		st = append(st, a.Vehicle.Drag)
		// Update Cr
		a.Vehicle.Drag = s[7]
		latestVector = mat64.NewVector(7, st)
	} else {
		latestVector = mat64.NewVector(6, s[0:6])
	}
	latestState := State{a.CurrentDT, *a.Vehicle, *a.Orbit, nil, latestVector}

	if a.computeSTM {
		// Extract the components of Φ
		rΦ, cΦ := a.perts.STMSize()
		sIdx := rΦ + 1
		ΦkTo0 := mat64.NewDense(rΦ, cΦ, nil)
		for i := 0; i < rΦ; i++ {
			for j := 0; j < cΦ; j++ {
				ΦkTo0.Set(i, j, s[sIdx])
				sIdx++
			}
		}
		// Compute the Φ for this transition
		var Φinv mat64.Dense
		if err := Φinv.Inverse(a.Φ); err != nil {
			panic(fmt.Errorf("could not invert the previous Φ: %s", err))
		}
		a.Φ.Mul(ΦkTo0, &Φinv)
		latestState.Φ = mat64.DenseCopyOf(a.Φ)
	}

	for _, histChan := range a.histChans {
		histChan <- latestState
	}

	// Let's execute any function which is in the queue of this time step.
	for _, f := range a.Vehicle.FuncQ {
		if f == nil {
			continue
		}
		f()
	}
	a.Vehicle.FuncQ = make([]func(), 5) // Clear the queue.

}

// Func is the integration function using Gaussian VOP as per Ruggiero et al. 2011.
func (a *Mission) Func(t float64, f []float64) (fDot []float64) {
	stateSize := 7
	if a.computeSTM {
		rSTM, cSTM := a.perts.STMSize()
		stateSize += rSTM * cSTM
		if a.perts.Drag {
			stateSize += 1
		}
	}
	fDot = make([]float64, stateSize) // init return vector
	// Let's add the thrust to increase the magnitude of the velocity.
	// XXX: Should this Accelerate call be with tmpOrbit?!
	Δv, usedFuel := a.Vehicle.Accelerate(a.CurrentDT, a.Orbit)
	var tmpOrbit *Orbit

	R := []float64{f[0], f[1], f[2]}
	V := []float64{f[3], f[4], f[5]}
	tmpOrbit = NewOrbitFromRV(R, V, a.Orbit.Origin)
	bodyAcc := -tmpOrbit.Origin.μ / math.Pow(Norm(R), 3)
	_, _, i, Ω, _, _, _, _, u := tmpOrbit.Elements()
	Δv = Rot313Vec(-u, -i, -Ω, Δv)
	// d\vec{R}/dt
	fDot[0] = f[3]
	fDot[1] = f[4]
	fDot[2] = f[5]
	// d\vec{V}/dt
	fDot[3] = bodyAcc*f[0] + Δv[0]
	fDot[4] = bodyAcc*f[1] + Δv[1]
	fDot[5] = bodyAcc*f[2] + Δv[2]
	// d(fuel)/dt
	fDot[6] = -usedFuel

	// Compute and add the perturbations (which are method dependent).
	pert := a.perts.Perturb(*tmpOrbit, a.CurrentDT, *a.Vehicle)

	// Compute STM if needed.
	if a.computeSTM {
		// Extract the components of Φ
		rΦ, cΦ := a.perts.STMSize()
		fIdx := rΦ + 1
		Φ := mat64.NewDense(rΦ, cΦ, nil)
		ΦDot := mat64.NewDense(rΦ, cΦ, nil)
		for i := 0; i < rΦ; i++ {
			for j := 0; j < cΦ; j++ {
				Φ.Set(i, j, f[fIdx])
				fIdx++
			}
		}

		// Compute the STM.
		A := mat64.NewDense(rΦ, cΦ, nil)
		// Top right is Identity 3x3
		A.Set(0, 3, 1)
		A.Set(1, 4, 1)
		A.Set(2, 5, 1)
		if rΦ == 7 {
			A.Set(3, 6, 1)
		}
		// Bottom left is where the magic is.
		x := R[0]
		y := R[1]
		z := R[2]
		x2 := math.Pow(R[0], 2)
		y2 := math.Pow(R[1], 2)
		z2 := math.Pow(R[2], 2)
		r2 := x2 + y2 + z2
		r232 := math.Pow(r2, 3/2.)
		r252 := math.Pow(r2, 5/2.)

		// Add the body perturbations
		dAxDx := 3*a.Orbit.Origin.μ*x2/r252 - a.Orbit.Origin.μ/r232
		dAxDy := 3 * a.Orbit.Origin.μ * x * y / r252
		dAxDz := 3 * a.Orbit.Origin.μ * x * z / r252
		dAyDx := 3 * a.Orbit.Origin.μ * x * y / r252
		dAyDy := 3*a.Orbit.Origin.μ*y2/r252 - a.Orbit.Origin.μ/r232
		dAyDz := 3 * a.Orbit.Origin.μ * y * z / r252
		dAzDx := 3 * a.Orbit.Origin.μ * x * z / r252
		dAzDy := 3 * a.Orbit.Origin.μ * y * z / r252
		dAzDz := 3*a.Orbit.Origin.μ*z2/r252 - a.Orbit.Origin.μ/r232

		A.Set(3, 0, dAxDx)
		A.Set(4, 0, dAyDx)
		A.Set(5, 0, dAzDx)
		A.Set(3, 1, dAxDy)
		A.Set(4, 1, dAyDy)
		A.Set(5, 1, dAzDy)
		A.Set(3, 2, dAxDz)
		A.Set(4, 2, dAyDz)
		A.Set(5, 2, dAzDz)

		// Jn perturbations:
		if a.perts.Jn > 1 {
			// Ai0 = \frac{\partial a}{\partial x}
			// Ai1 = \frac{\partial a}{\partial y}
			// Ai2 = \frac{\partial a}{\partial z}
			A30 := A.At(3, 0)
			A40 := A.At(4, 0)
			A50 := A.At(5, 0)
			A31 := A.At(3, 1)
			A41 := A.At(4, 1)
			A51 := A.At(5, 1)
			A32 := A.At(3, 2)
			A42 := A.At(4, 2)
			A52 := A.At(5, 2)

			// Notation simplification
			z3 := math.Pow(R[2], 3)
			z4 := math.Pow(R[2], 4)
			// Adding those fractions to avoid forgetting the trailing period which makes them floats.
			f32 := 3 / 2.
			f152 := 15 / 2.
			r272 := math.Pow(r2, 7/2.)
			r292 := math.Pow(r2, 9/2.)
			// J2
			j2fact := a.Orbit.Origin.J(2) * math.Pow(a.Orbit.Origin.Radius, 2) * a.Orbit.Origin.μ
			A30 += -f32 * j2fact * (35*x2*z2/r292 - 5*x2/r272 - 5*z2/r272 + 1/r252) //dAxDx
			A40 += -f152 * j2fact * (7*x*y*z2/r292 - x*y/r272)                      //dAyDx
			A50 += -f152 * j2fact * (7*x*z3/r292 - 3*x*z/r272)                      //dAzDx

			A31 += -f152 * j2fact * (7*x*y*z2/r292 - x*y/r272)                      //dAxDy
			A41 += -f32 * j2fact * (35*y2*z2/r292 - 5*y2/r272 - 5*z2/r272 + 1/r252) // dAyDy
			A51 += -f152 * j2fact * (7*y*z3/r292 - 3*y*z/r272)                      // dAzDy

			A32 += -f152 * j2fact * (7*x*z3/r292 - 3*x*z/r272)        //dAxDz
			A42 += -f152 * j2fact * (7*y*z3/r292 - 3*y*z/r272)        //dAyDz
			A52 += -f32 * j2fact * (35*z4/r292 - 30*z2/r272 + 3/r252) // dAzDz

			// J3
			if a.perts.Jn > 2 {
				z5 := math.Pow(R[2], 5)
				r2112 := math.Pow(r2, 11/2.)
				f52 := 5 / 2.
				f1052 := 105 / 2.
				j3fact := a.Orbit.Origin.J(3) * math.Pow(a.Orbit.Origin.Radius, 3) * a.Orbit.Origin.μ
				A30 += -f52 * j3fact * (63*x2*z3/r2112 - 21*x2*z/r292 - 7*z3/r292 + 3*z/r272) //dAxDx
				A40 += -f1052 * j3fact * (3*x*y*z3/r2112 - x*y*z/r292)                        //dAyDx
				A50 += -f152 * j3fact * (21*x*z4/r2112 - 14*x*z2/r292 + x/r272)               //dAzDx

				A31 += -f1052 * j3fact * (3*x*y*z3/r2112 - x*y*z/r292)                        //dAxDy
				A41 += -f52 * j3fact * (63*y2*z3/r2112 - 21*y2*z/r292 - 7*z3/r292 + 3*z/r272) // dAyDy
				A51 += -f152 * j3fact * (21*y*z4/r2112 - 14*y*z2/r292 + y/r272)               // dAzDy

				A32 += -f152 * j3fact * (21*x*z4/r2112 - 14*x*z2/r292 + x/r272) //dAxDz
				A42 += -f152 * j3fact * (21*y*z4/r2112 - 14*y*z2/r292 + y/r272) //dAyDz
				A52 += -f52 * j3fact * (63*z5/r2112 - 70*z3/r292 + 15*z/r272)   // dAzDz
			}
			// \frac{\partial a}{\partial x}
			A.Set(3, 0, A30)
			A.Set(4, 0, A40)
			A.Set(5, 0, A50)
			// \partial a/\partial y
			A.Set(3, 1, A31)
			A.Set(4, 1, A41)
			A.Set(5, 1, A51)
			// \partial a/\partial z
			A.Set(3, 2, A32)
			A.Set(4, 2, A42)
			A.Set(5, 2, A52)
		}

		var RSunToEarth, RSunToSC, REarthToSC []float64

		if a.perts.Drag || a.perts.PerturbingBody != nil {
			REarthToSC = a.Orbit.R()
			RSunToEarth = MxV33(R1(Deg2rad(-Earth.tilt)), a.Orbit.Origin.HelioOrbit(a.CurrentDT).R())
			RSunToSC = make([]float64, 3)
			for i := 0; i < 3; i++ {
				RSunToSC[i] = RSunToEarth[i] + REarthToSC[i]
			}
		}

		// BUG: This includes BOTH SRP and Sun perturbations.
		if a.perts.Drag {
			Cr := a.Vehicle.Drag
			S := 0.01e-6 // TODO: Idem for the Area to mass ratio
			Phi := 1357.
			// Build the vectors.
			celerity := 2.997925e+05
			srpCst := -Sun.μ + (Phi*AU*AU*S/celerity)*Cr
			RSunToSC3 := math.Pow(Norm(RSunToSC), 3)
			RSunToSC5 := math.Pow(Norm(RSunToSC), 5)

			// Getting values
			// Ai0 = \frac{\partial a}{\partial x}
			// Ai1 = \frac{\partial a}{\partial y}
			// Ai2 = \frac{\partial a}{\partial z}
			A30 := A.At(3, 0)
			A40 := A.At(4, 0)
			A50 := A.At(5, 0)
			A31 := A.At(3, 1)
			A41 := A.At(4, 1)
			A51 := A.At(5, 1)
			A32 := A.At(3, 2)
			A42 := A.At(4, 2)
			A52 := A.At(5, 2)

			dAxDx := srpCst/RSunToSC3 + srpCst*(-1.5/RSunToSC5)*(-RSunToSC[0])*2*(-RSunToSC[0])
			dAxDy := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[0]) * 2 * (-RSunToSC[1])
			dAxDz := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[0]) * 2 * (-RSunToSC[2])
			dAxDCr := (Phi * AU * AU * S / celerity) / (RSunToSC3 * (-RSunToSC[0]))
			dAyDx := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[1]) * 2 * (-RSunToSC[0])
			dAyDy := srpCst/RSunToSC3 + srpCst*(-1.5/RSunToSC5)*(-RSunToSC[1])*2*(-RSunToSC[1])
			dAyDz := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[1]) * 2 * (-RSunToSC[2])
			dAyDCr := (Phi * AU * AU * S / celerity) / (RSunToSC3 * (-RSunToSC[1]))
			dAzDx := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[2]) * 2 * (-RSunToSC[0])
			dAzDy := srpCst * (-1.5 / RSunToSC5) * (-RSunToSC[2]) * 2 * (-RSunToSC[1])
			dAzDz := srpCst/RSunToSC3 + srpCst*(-1.5/RSunToSC5)*(-RSunToSC[2])*2*(-RSunToSC[2])
			dAzDCr := (Phi * AU * AU * S / celerity) / (RSunToSC3 * (-RSunToSC[2]))
			// Setting values
			// \frac{\partial a}{\partial x}
			A.Set(3, 0, A30+dAxDx)
			A.Set(4, 0, A40+dAyDx)
			A.Set(5, 0, A50+dAzDx)
			// \partial a/\partial y
			A.Set(3, 1, A31+dAxDy)
			A.Set(4, 1, A41+dAyDy)
			A.Set(5, 1, A51+dAzDy)
			// \partial a/\partial z
			A.Set(3, 2, A32+dAxDz)
			A.Set(4, 2, A42+dAyDz)
			A.Set(5, 2, A52+dAzDz)
			// \partial a/\partial Cr
			A.Set(3, 6, dAxDCr)
			A.Set(4, 6, dAyDCr)
			A.Set(5, 6, dAzDCr)
		}

		ΦDot.Mul(A, Φ)

		// Store ΦDot in fDot
		fIdx = rΦ + 1
		if a.perts.Drag {
			fDot[fIdx-1] = a.Vehicle.Drag
		}
		for i := 0; i < rΦ; i++ {
			for j := 0; j < cΦ; j++ {
				fDot[fIdx] = ΦDot.At(i, j)
				fIdx++
			}
		}
	}

	// Sanity check
	for i := 0; i < stateSize; i++ {
		if i < 7 {
			fDot[i] += pert[i]
		}
		if math.IsNaN(fDot[i]) {
			r, v := a.Orbit.RV()
			panic(fmt.Errorf("fDot[%d]=NaN @ dt=%s\ncur:%s\tΔv=%+v\nR=%+v\tV=%+v", i, a.CurrentDT, a.Orbit, Δv, r, v))
		}
	}
	return
}

// State stores propagated state.
type State struct {
	DT      time.Time
	SC      Spacecraft
	Orbit   Orbit
	Φ       *mat64.Dense // STM
	cVector *mat64.Vector
}

// Vector returns the orbit vector with position and velocity.
func (s State) Vector() *mat64.Vector {
	if s.cVector == nil {
		var vec *mat64.Vector
		if s.SC.Drag > 0 {
			vec = mat64.NewVector(7, nil)
			vec.SetVec(6, s.SC.Drag)
		} else {
			vec = mat64.NewVector(6, nil)
		}
		R, V := s.Orbit.RV()
		for i := 0; i < 3; i++ {
			vec.SetVec(i, R[i])
			vec.SetVec(i+3, V[i])
		}
		s.cVector = vec
	}
	return s.cVector
}
