package main

import (
        "fmt"
        "math"
"os"
        "github.com/ChristopherRabotin/smd"
)

func main(){
  rSOI := []float64{546507.344255845,-527978.380486028,531109.066836708}
  vSOI := []float64{-4.9220589268733 ,5.36316523097915 ,-5.22166308425181}
  // Compute nominal values
  bRStar, bTStar, _:= smd.NewOrbitFromRV(rSOI, vSOI, smd.Earth).BPlane()
  csv := fmt.Sprintf("perturbation\tdBt/dVx\tdBr/dVx\tdBt/dVy\tdBr/dVy\n")
  for pertExp := -15.; pertExp < 3; pertExp ++{
    for fact := 1.; fact < 10; fact += 0.05{
      Δv := fact*math.Pow(10, pertExp)
      csv += fmt.Sprintf("%.18f\t", Δv)
      for i := 0;i<2;i++{
        vSOItmp := make([]float64, 3)
        copy(vSOItmp, vSOI)
        vSOItmp[i] += Δv
        orbit := smd.NewOrbitFromRV(rSOI, vSOItmp, smd.Earth)
        bR, bT , _ := orbit.BPlane()
        dbR := (bR-bRStar)/Δv
        dbT := (bT-bTStar)/Δv
        csv += fmt.Sprintf("%.18f\t%.18f\t", dbT, dbR)
      }
      csv += fmt.Sprintf("\n")
    }
  }
  // Write CSV file.
  f, err := os.Create("./bplane-perts")
  if err != nil {
    panic(err)
  }
  defer f.Close()
  if _, err := f.WriteString(csv); err != nil {
    panic(err)
  }
}
