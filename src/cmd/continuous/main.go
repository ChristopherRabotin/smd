package main

import (
	"dynamics"
	"fmt"
	"math"
	"os"
	"runtime"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	//end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.
	end := time.Date(2018, 1, 3, 0, 0, 0, 0, time.UTC)

	/* Let's propagate out of Mars at a guessed date of 7 months after launch date from Earth.
	Note that we only output the CSV because we don't need to visualize this.
	* /
	startM := time.Date(2016, 10, 10, 0, 0, 0, 0, time.UTC)
	endM := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	scMars := SpacecraftFromMars("IM")
	scMars.LogInfo()
	astroM := dynamics.NewAstro(scMars, InitialMarsOrbit(), startM, endM, dynamics.ExportConfig{Filename: "IM", OE: false, Cosmo: false, Timestamp: false})
	astroM.Propagate()

	target := astroM.Orbit*/
	target := dynamics.NewOrbitFromOE(226090298.679, 0.088, 26.195, 3.516, 326.494, 278.358, dynamics.Sun)
	fmt.Printf("target orbit: %s\n", target)
	sc := SpacecraftFromEarth("IE", *target)
	sc.LogInfo()
	astro := dynamics.NewAstro(sc, InitialEarthOrbit(), start, end, dynamics.ExportConfig{Filename: "IE", OE: true, Cosmo: true, Timestamp: false})
	astro.Propagate()

}

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}
