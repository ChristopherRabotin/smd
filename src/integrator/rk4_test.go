package integrator

import (
	"dynamics"
	"fmt"
	"math"
	"testing"
)

type Balbasi1D struct {
	state  []float64 // Note that we don't have a state history here.
	prevIt uint
}

func NewBalbasi1D() (b *Balbasi1D) {
	b = &Balbasi1D{}
	b.state = []float64{1200.0}
	b.prevIt = 0
	return
}

func (b *Balbasi1D) GetState() []float64 {
	return b.state
}

func (b *Balbasi1D) SetState(i uint64, s []float64) {
	b.state = s
	if i != 0 && b.prevIt+1 != uint(i) {
		panic(fmt.Errorf("expected i=%d, got i=%d", b.prevIt+1, i))
	}
	b.prevIt = uint(i)
}

func (b *Balbasi1D) Stop(i uint64) bool {
	return i*30 >= 480
}

func (b *Balbasi1D) Func(t float64, s []float64) []float64 {
	val := []float64{(-2.2067 * 1e-12) * (math.Pow(s[0], 4) - 81*1e8)}
	return val
}

func TestRK4In1D(t *testing.T) {
	inte := NewBalbasi1D()
	if _, _, err := NewRK4(1, 30, inte).Solve(); err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	if diff := math.Abs(inte.GetState()[0] - 647.5720536920); diff >= 1e-10 {
		t.Fatalf("expected final state %4.10f is different by %4.10f", inte.GetState()[0], diff)
	}
}

// AttitudeTest tests that the energy is conserved for a given body.
type AttitudeTest struct {
	*dynamics.Attitude
}

func (a *AttitudeTest) Stop(i uint64) bool {
	return float64(i)*1e-6 >= 1e-1
}

func NewAttitudeTest() (a *AttitudeTest) {
	a = &AttitudeTest{dynamics.NewAttitude([3]float64{0.3, -0.4, 0.5}, [3]float64{0.1, 0.4, -0.2},
		[]float64{10.0, 0, 0, 0, 5.0, 0, 0, 0, 2.0})}
	return
}

func TestRK4Attitude(t *testing.T) {
	inte := NewAttitudeTest()
	initMom := inte.Momentum()
	if _, _, err := NewRK4(0, 1e-6, inte).Solve(); err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	if diff := math.Abs(initMom - inte.Momentum()); diff > 1e-8 {
		t.Fatalf("angular momentum changed by %4.12f", diff)
	}
}
