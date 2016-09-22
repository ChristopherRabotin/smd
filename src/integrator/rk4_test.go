package integrator

import (
	"fmt"
	"math"
	"math/big"
	"testing"
)

type Balbasi1D struct {
	state []big.Float // Note that we don't have a state history here.
}

func NewBalbasi1D() (b *Balbasi1D) {
	b = &Balbasi1D{}
	b.state = []big.Float{*big.NewFloat(1200.0)}
	return
}

func (b *Balbasi1D) GetState() []big.Float {
	return b.state
}

func (b *Balbasi1D) SetState(i uint64, s []big.Float) {
	b.state = s
	state, _ := b.GetState()[0].Float64()
	fmt.Printf("Updated state to %5.5f\n", state)
}

func (b *Balbasi1D) Stop(i uint64) bool {
	return i*30 >= 480
}

func (b *Balbasi1D) Func(t *big.Float, s []big.Float) []big.Float {
	rtn := make([]big.Float, 1)
	theta, _ := s[0].Float64()
	fmt.Printf("theta = %3.5f\t", theta)
	val := (-2.2067 * 1e-12) * (math.Pow(theta, 4) - 81*1e8)
	fmt.Printf("rtn[0] = %3.5f\n", val)
	rtn[0] = *big.NewFloat(val)
	return rtn
}

func TestRK4In1D(t *testing.T) {
	t.Log("Starting test")
	BConfig := &Config{X0: big.NewFloat(1), StepSize: 30, Integator: NewBalbasi1D()}
	iterNum, xi, err := SolveRK4(BConfig)
	if err != nil {
		t.Fatalf("err: %+v\n", err)
	}
	fmt.Printf("iterNum = %+v\nxi =%+v\n", iterNum, xi.String())
	state, _ := BConfig.Integator.GetState()[0].Float64()
	fmt.Printf("Final state = %5.5f\n", state)
}
