package integrator

import (
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
		t.Fatalf("expected final state is different by %4.10f", diff)
	}
}
