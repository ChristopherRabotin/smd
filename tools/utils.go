package tools

import "github.com/gonum/matrix/mat64"

func norm(v *mat64.Vector) float64 {
	return mat64.Norm(v, 2)
}
