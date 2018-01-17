package smd

import (
	"fmt"
	"math"
	"time"

	"github.com/gonum/floats"
)

// ControlLaw defines an enum of control laws.
type ControlLaw uint8

// ControlLawType defines the way to sum different Lyuapunov optimal CL
type ControlLawType uint8

type hohmannStatus uint8

const (
	tangential ControlLaw = iota + 1
	antiTangential
	coast
	multiOpti
	hohmann
	// OptiΔaCL allows to optimize thrust for semi major axis change
	OptiΔaCL
	// OptiΔiCL allows to optimize thrust for inclination change
	OptiΔiCL
	// OptiΔeCL allows to optimize thrust for eccentricity change
	OptiΔeCL
	// OptiΔΩCL allows to optimize thrust forRAAN change
	OptiΔΩCL
	// OptiΔωCL allows to optimize thrust for argument of perigee change
	OptiΔωCL
	// Ruggiero uses the eponym method of combining the control laws
	Ruggiero ControlLawType = iota + 1
	// Naasz is another type of combination of control law
	Naasz
	hohmannCompute hohmannStatus = iota + 1
	hohmmanInitΔv
	hohmmanFinalΔv
	hohmmanCoast
	hohmmanCompleted
)

func (cl ControlLaw) String() string {
	switch cl {
	case tangential:
		return "tan"
	case antiTangential:
		return "aTan"
	case coast:
		return "coast"
	case OptiΔaCL:
		return "optiΔa"
	case OptiΔeCL:
		return "optiΔe"
	case OptiΔiCL:
		return "optiΔi"
	case OptiΔΩCL:
		return "optiΔΩ"
	case OptiΔωCL:
		return "optiΔω"
	case multiOpti:
		return "multiOpti"
	case hohmann:
		return "Hohmann"
	}
	panic("cannot stringify unknown control law")
}

func (meth ControlLawType) String() string {
	switch meth {
	case Ruggiero:
		return "Ruggiero"
	case Naasz:
		return "Naasz"
	}
	panic("cannot stringify unknown control law summation method")
}

// ThrustControl defines a thrust control interface.
type ThrustControl interface {
	Control(o Orbit) []float64
	Type() ControlLaw
	Reason() string
}

// GenericCL partially defines a ThrustControl.
type GenericCL struct {
	reason string
	cl     ControlLaw
}

// Reason implements the ThrustControl interface.
func (cl GenericCL) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl GenericCL) Type() ControlLaw {
	return cl.cl
}

func newGenericCLFromCL(cl ControlLaw) GenericCL {
	return GenericCL{cl.String(), cl}
}

/* Let's define some control laws. */

// Coast defines an thrust control law which does not thrust.
type Coast struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl Coast) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl Coast) Type() ControlLaw {
	return coast
}

// Control implements the ThrustControl interface.
func (cl Coast) Control(o Orbit) []float64 {
	return []float64{0, 0, 0}
}

// Tangential defines a tangential thrust control law
type Tangential struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl Tangential) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl Tangential) Type() ControlLaw {
	return tangential
}

// Control implements the ThrustControl interface.
func (cl Tangential) Control(o Orbit) []float64 {
	return []float64{0, 1, 0}
}

// AntiTangential defines an antitangential thrust control law
type AntiTangential struct {
	reason string
}

// Reason implements the ThrustControl interface.
func (cl AntiTangential) Reason() string {
	return cl.reason
}

// Type implements the ThrustControl interface.
func (cl AntiTangential) Type() ControlLaw {
	return antiTangential
}

// Control implements the ThrustControl interface.
func (cl AntiTangential) Control(o Orbit) []float64 {
	return []float64{0, -1, 0}
}

/* Following optimal thrust change are from IEPC 2011's paper:
Low-Thrust Maneuvers for the Efficient Correction of Orbital Elements
A. Ruggiero, S. Marcuccio and M. Andrenucci */

func unitΔvFromAngles(α, β float64) []float64 {
	sinα, cosα := math.Sincos(α)
	sinβ, cosβ := math.Sincos(β)
	return []float64{sinα * cosβ, cosα * cosβ, sinβ}
}

func anglesFromUnitΔv(Δv []float64) (α, β float64) {
	β = math.Asin(Δv[2])
	cosβ := math.Cos(β)
	α = math.Asin(Δv[0] / cosβ)
	return
}

// OptimalThrust is an optimal thrust.
type OptimalThrust struct {
	ctrl func(o Orbit) []float64
	GenericCL
}

// Control implements the ThrustControl interface.
func (cl OptimalThrust) Control(o Orbit) []float64 {
	return cl.ctrl(o)
}

// NewOptimalThrust returns a new optimal Δe.
func NewOptimalThrust(cl ControlLaw, reason string) ThrustControl {
	var ctrl func(o Orbit) []float64
	switch cl {
	case OptiΔaCL:
		ctrl = func(o Orbit) []float64 {
			_, e, _, _, _, ν, _, _, _ := o.Elements()
			sinν, cosν := math.Sincos(ν)
			return unitΔvFromAngles(math.Atan2(e*sinν, 1+e*cosν), 0.0)
		}
		break
	case OptiΔeCL:
		ctrl = func(o Orbit) []float64 {
			_, cosE := o.SinCosE()
			_, _, _, _, _, ν, _, _, _ := o.Elements()
			sinν, cosν := math.Sincos(ν)
			return unitΔvFromAngles(math.Atan2(sinν, cosν+cosE), 0.0)
		}
		break
	case OptiΔiCL:
		ctrl = func(o Orbit) []float64 {
			_, _, _, _, ω, ν, _, _, _ := o.Elements()
			return unitΔvFromAngles(0.0, Sign(math.Cos(ω+ν))*math.Pi/2)
		}
		break
	case OptiΔΩCL:
		ctrl = func(o Orbit) []float64 {
			_, _, _, _, ω, ν, _, _, _ := o.Elements()
			return unitΔvFromAngles(0.0, Sign(math.Sin(ω+ν))*math.Pi/2)
		}
		break
	case OptiΔωCL:
		// The argument of periapsis control is from Petropoulos and in plane.
		// The out of plane will change other orbital elements at the same time.
		// We determine which one to use based on the efficiency of each.
		ctrl = func(o Orbit) []float64 {
			_, e, i, _, ω, ν, _, _, _ := o.Elements()
			oe2 := 1 - math.Pow(e, 2)
			e3 := math.Pow(e, 3)
			νOptiα := math.Acos(math.Pow(oe2/(2*e3)+math.Sqrt(0.25*math.Pow(oe2/e3, 2)+1/27.), 1/3.) - math.Pow(-oe2/(2*e3)+math.Sqrt(0.25*math.Pow(oe2/e3, 2)+1/27.), 1/3.) - 1/e)
			νOptiβ := math.Acos(-e*math.Cos(ω)) - ω
			if math.Abs(ν-νOptiα) < math.Abs(ν-νOptiβ) {
				// The true anomaly is closer to the optimal in plane thrust, so let's do an in-plane thrust.
				p := o.SemiParameter()
				sinν, cosν := math.Sincos(ν)
				return unitΔvFromAngles(math.Atan2(-p*cosν, (p+o.RNorm())*sinν), 0.0)
			}
			return unitΔvFromAngles(0.0, Sign(-math.Sin(ω+ν))*math.Cos(i)*math.Pi/2)
		}
		break
	default:
		panic(fmt.Errorf("optmized %s not yet implemented", cl))
	}
	return OptimalThrust{ctrl, GenericCL{reason, cl}}
}

// OptimalΔOrbit combines all the control laws from Ruggiero et al.
type OptimalΔOrbit struct {
	Initd, cleared bool
	controls       []ThrustControl
	method         ControlLawType
	// local copy of the OEs of the inital and target orbits
	oInita, oInite, oIniti, oInitΩ, oInitω, oInitν float64
	oTgta, oTgte, oTgti, oTgtΩ, oTgtω, oTgtν       float64
	Distanceε, Eccentricityε, Angleε               float64
	GenericCL
}

// NewOptimalΔOrbit generates a new OptimalΔOrbit based on the provided target orbit.
func NewOptimalΔOrbit(target Orbit, method ControlLawType, laws []ControlLaw) *OptimalΔOrbit {
	cl := OptimalΔOrbit{}
	cl.cleared = false
	cl.method = method
	cl.oTgta, cl.oTgte, cl.oTgti, cl.oTgtΩ, cl.oTgtω, cl.oTgtν, _, _, _ = target.Elements()
	cl.Distanceε = distanceε
	cl.Eccentricityε = eccentricityε
	cl.Angleε = angleε
	if len(laws) == 0 {
		laws = []ControlLaw{OptiΔaCL, OptiΔeCL, OptiΔiCL, OptiΔΩCL, OptiΔωCL}
	}
	cl.controls = make([]ThrustControl, len(laws))
	for i, law := range laws {
		cl.controls[i] = NewOptimalThrust(law, law.String())
	}
	if len(cl.controls) > 1 {
		cl.GenericCL = GenericCL{"ΔOrbit", multiOpti}
	} else {
		cl.GenericCL = GenericCL{"ΔOrbit", cl.controls[0].Type()}
	}
	return &cl
}

// SetTarget changes the target of this optimal control
func (cl *OptimalΔOrbit) SetTarget(target Orbit) {
	cl.oTgta, cl.oTgte, cl.oTgti, cl.oTgtΩ, cl.oTgtω, cl.oTgtν, _, _, _ = target.Elements()
}

// SetEpsilons changes the target of this optimal control
func (cl *OptimalΔOrbit) SetEpsilons(distanceε, eccentricityε, angleε float64) {
	cl.Distanceε = distanceε
	cl.Eccentricityε = eccentricityε
	cl.Angleε = angleε
}

func (cl *OptimalΔOrbit) String() string {
	return "OptimalΔOrbit"
}

// Control implements the ThrustControl interface.
func (cl *OptimalΔOrbit) Control(o Orbit) []float64 {
	thrust := []float64{0, 0, 0}
	if !cl.Initd {
		cl.Initd = true
		cl.oInita, cl.oInite, cl.oIniti, cl.oInitΩ, cl.oInitω, cl.oInitν, _, _, _ = o.Elements()
		if len(cl.controls) == 5 {
			// Let's populate this with the appropriate laws, so we're resetting it.
			cl.controls = make([]ThrustControl, 0)
			if !floats.EqualWithinAbs(cl.oInita, cl.oTgta, cl.Distanceε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔaCL, "Δa"))
			}
			if !floats.EqualWithinAbs(cl.oInite, cl.oTgte, cl.Eccentricityε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔeCL, "Δe"))
			}
			if !floats.EqualWithinAbs(cl.oIniti, cl.oTgti, cl.Angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔiCL, "Δi"))
			}
			if !floats.EqualWithinAbs(cl.oInitΩ, cl.oTgtΩ, cl.Angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔΩCL, "ΔΩ"))
			}
			if !floats.EqualWithinAbs(cl.oInitω, cl.oTgtω, cl.Angleε) {
				cl.controls = append(cl.controls, NewOptimalThrust(OptiΔωCL, "Δω"))
			}
		}
		return thrust
	}

	cl.cleared = true // Will be set to false if not yet converged.
	a, e, i, Ω, ω, _, _, _, _ := o.Elements()
	switch cl.method {
	case Ruggiero:
		factor := func(oscul, init, target, tol float64) float64 {
			if floats.EqualWithinAbs(oscul, target, tol) {
				return 0
			}
			if floats.EqualWithinAbs(init, target, tol) {
				init += tol // Adding a small error to avoid NaN while still making the correction
			}
			return (target - oscul) / math.Abs(target-init)
		}

		for _, ctrl := range cl.controls {
			var oscul, init, target, tol float64
			switch ctrl.Type() {
			case OptiΔaCL:
				oscul = a
				init = cl.oInita
				target = cl.oTgta
				tol = cl.Distanceε
			case OptiΔeCL:
				oscul = e
				init = cl.oInite
				target = cl.oTgte
				tol = cl.Eccentricityε
			case OptiΔiCL:
				oscul = i
				init = cl.oIniti
				target = cl.oTgti
				tol = cl.Angleε
			case OptiΔΩCL:
				oscul = Ω
				init = cl.oInitΩ
				target = cl.oTgtΩ
				tol = cl.Angleε
			case OptiΔωCL:
				oscul = ω
				init = cl.oInitω
				target = cl.oTgtω
				tol = cl.Angleε
			}
			// XXX: This summation may be wrong: |\sum x_i| != \sum |x_i|.
			if fact := factor(oscul, init, target, tol); fact != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				for i := 0; i < 3; i++ {
					thrust[i] += fact * tmpThrust[i]
				}
			}
		}
	case Naasz:
		// Note that, as described in Hatten MSc. thesis, the summing method only
		// works one way (because of the δO^2) per OE. So I added the sign function
		// to fix it.
		//dε, eε, aε := o.epsilons()
		for _, ctrl := range cl.controls {
			var weight, δO float64
			p := o.SemiParameter()
			h := o.HNorm()
			sinω, cosω := math.Sincos(ω)
			switch ctrl.Type() {
			case OptiΔaCL:
				δO = cl.oTgta - a
				if math.Abs(δO) < cl.Distanceε {
					δO = 0
				}
				weight = Sign(δO) * math.Pow(h, 2) / (4 * math.Pow(a, 4) * math.Pow(1+e, 2))
			case OptiΔeCL:
				δO = cl.oTgte - e
				if math.Abs(δO) < cl.Eccentricityε {
					δO = 0
				}
				weight = Sign(δO) * math.Pow(h, 2) / (4 * math.Pow(p, 2))
			case OptiΔiCL:
				δO = cl.oTgti - i
				if math.Abs(δO) < cl.Angleε {
					δO = 0
				}
				weight = Sign(δO) * math.Pow((h+e*h*math.Cos(ω+math.Asin(e*sinω)))/(p*(math.Pow(e*sinω, 2)-1)), 2)
			case OptiΔΩCL:
				δO = cl.oTgtΩ - Ω
				if δO > math.Pi {
					// Enforce short path to correct angle.
					δO *= -1
				}
				if math.Abs(δO) < cl.Angleε {
					δO = 0
				}
				weight = Sign(δO) * math.Pow((h*math.Sin(i)*(e*math.Sin(ω+math.Asin(e*cosω))-1))/(p*(1-math.Pow(e*cosω, 2))), 2)
			case OptiΔωCL:
				δO = cl.oTgtω - ω
				if δO > math.Pi {
					// Enforce short path to correct angle.
					δO *= -1
				}
				if math.Abs(δO) < cl.Angleε {
					δO = 0
				}
				weight = Sign(δO) * (math.Pow(e*h, 2) / (4 * math.Pow(p, 2))) * (1 - math.Pow(e, 2)/4)
			}
			if δO != 0 {
				cl.cleared = false // We're not actually done.
				tmpThrust := ctrl.Control(o)
				fact := 0.5 * weight * math.Pow(δO, 2)
				for i := 0; i < 3; i++ {
					thrust[i] += fact * tmpThrust[i]
				}
			}
		}
	default:
		panic(fmt.Errorf("control law sumation %+v not yet supported", cl.method))
	}

	return Unit(thrust)
}

// HohmannΔv computes the Δv needed to go from one orbit to another, and performs an instantaneous Δv.
type HohmannΔv struct {
	target                      Orbit
	status                      hohmannStatus
	ΔvBurnInit, ΔvInit, ΔvFinal float64
	tof                         time.Duration
	GenericCL
}

// Precompute computes and displays the Hohmann transfer orbit.
func (cl *HohmannΔv) Precompute(o Orbit) {
	_, e, i, _, _, ν, _, _, _ := o.Elements()
	_, _, iTgt, _, _, νTgt, _, _, _ := cl.target.Elements()
	if !floats.EqualWithinAbs(νTgt, ν, angleε) && !floats.EqualWithinAbs(νTgt, ν+math.Pi, angleε) && !floats.EqualWithinAbs(νTgt, ν-math.Pi, angleε) {
		panic(fmt.Errorf("cannot perform Hohmann between orbits with misaligned semi-major axes\nini: %s\ntgt: %s", o, cl.target))
	}
	if !floats.EqualWithinAbs(e, 0, eccentricityε) {
		panic(fmt.Errorf("cannot perform Hohmann from a non elliptical orbit"))
	}
	if !floats.EqualWithinAbs(iTgt, i, angleε) {
		panic(fmt.Errorf("cannot perform Hohmann between non co-planar orbits\nini: %s\ntgt: %s", o, cl.target))
	}
	if !floats.EqualWithinAbs(ν, 0, angleε) && !floats.EqualWithinAbs(ν, math.Pi, angleε) {
		fmt.Printf("[WARNING] Hohmann transfer started neither at apoapsis nor at periapasis (inefficient)\n")
	}
	rInit := o.RNorm()
	rFinal := cl.target.RNorm()
	vInit := o.VNorm()
	vFinal := cl.target.VNorm()
	vDeparture, vArrival, tof := Hohmann(rInit, vInit, rFinal, vFinal, o.Origin)
	cl.ΔvInit = vDeparture - vInit
	cl.ΔvFinal = vArrival - vFinal
	cl.tof = tof
	durStr := cl.tof.String() + fmt.Sprintf(" (~%.1fd)", cl.tof.Hours()/24)
	fmt.Printf("=== HOHMANN TRANSFER INFO ===\nHohmann transfer information - T.O.F.: %s\nvInit=%f km/s\tvFinal=%f km/s\nvDeparture=%f km/s\t vArrival=%f km/s\nΔvInit=%f km/s\tΔvFinal=%f\n=== HOHMANN TRANSFER END ====\n", durStr, vInit, vFinal, vDeparture, vArrival, cl.ΔvInit, cl.ΔvFinal)
}

// Control implements the ThrustControl interface.
func (cl *HohmannΔv) Control(o Orbit) []float64 {

	switch cl.status {
	case hohmmanCoast:
		fallthrough
	case hohmmanCompleted:
		return []float64{0, 0, 0}
	case hohmmanInitΔv:
		if floats.EqualWithinAbs(cl.ΔvBurnInit-o.VNorm(), cl.ΔvInit, velocityε) {
			// We have applied enough Δv, so let's stop burning.
			cl.status = hohmmanCoast
			return []float64{0, 0, 0}
		}
		return []float64{Sign(cl.ΔvInit), 0, 0}
	case hohmmanFinalΔv:
		if floats.EqualWithinAbs(cl.ΔvBurnInit-o.VNorm(), cl.ΔvFinal, velocityε) {
			// We have applied enough Δv, so let's stop burning.
			cl.status = hohmmanCompleted
			cl.ΔvBurnInit = 0 // Reset to zero after burn is completed.
			return []float64{0, 0, 0}
		}
		return []float64{Sign(cl.ΔvFinal), 0, 0}
	default:
		panic("unreachable code")
	}
}

// NewHohmannΔv defines a new inversion control law.
func NewHohmannΔv(target Orbit) HohmannΔv {
	_, e, _, _, _, _, _, _, _ := target.Elements()
	if !floats.EqualWithinAbs(e, 0, eccentricityε) {
		panic(fmt.Errorf("cannot perform Hohmann to a non elliptical orbit"))
	}
	return HohmannΔv{target, hohmannCompute, 0, 0, 0, time.Duration(-1) * time.Second, newGenericCLFromCL(hohmann)}
}

// Maneuver stores a maneuver in the VNC frame
type Maneuver struct {
	R, N, C float64
	done    bool
}

// Δv returns the Δv in km/s
func (m Maneuver) Δv() float64 {
	return math.Sqrt(m.R*m.R + m.N*m.N + m.C*m.C)
}

func (m Maneuver) String() string {
	return fmt.Sprintf("burn [%f %f %f] km/s -- executed: %v", m.R, m.N, m.C, m.done)
}

func NewManeuver(R, N, C float64) Maneuver {
	return Maneuver{R, N, C, false}
}
