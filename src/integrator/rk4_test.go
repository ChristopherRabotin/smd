package integrator

import (
	"fmt"
	"math"
	"testing"
)

type Balbasi1D struct {
	state []float64 // Note that we don't have a state history here.
}

func NewBalbasi1D() (b *Balbasi1D) {
	b = &Balbasi1D{}
	b.state = []float64{1200.0}
	return
}

func (b *Balbasi1D) GetState() []float64 {
	return b.state
}

func (b *Balbasi1D) SetState(i uint64, s []float64) {
	b.state = s
	fmt.Printf("Updated state to %5.5f\n", b.GetState()[0])
}

func (b *Balbasi1D) Stop(i uint64) bool {
	return i*30 >= 480
}

func (b *Balbasi1D) Func(t float64, s []float64) []float64 {
	fmt.Printf("theta = %3.5f\t", s[0])
	val := []float64{(-2.2067 * 1e-12) * (math.Pow(s[0], 4) - 81*1e8)}
	fmt.Printf("rtn[0] = %3.5f\n", val)
	return val
}

func TestRK4In1D(t *testing.T) {
	t.Log("Starting test")
	rk4 := NewRK4(1, 30, NewBalbasi1D())

	iterNum, xi, err := rk4.Solve()
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	fmt.Printf("iterNum = %+v\nxi =%+v\n", iterNum, xi)
	fmt.Printf("Final state = %5.5f\n", rk4.Integator.GetState()[0])
}
