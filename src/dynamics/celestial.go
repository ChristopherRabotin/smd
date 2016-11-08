package dynamics

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// AU is one astronomical unit in kilometers.
	AU = 149598000
)

// CelestialObject defines a celestial object.
// Note: globe and elements may be nil; does not support satellites yet.
type CelestialObject struct {
	Name   string
	Radius float64
	a      float64
	μ      float64
	tilt   float64 // Axial tilt
	SOI    float64 // With respect to the Sun
	J2     float64
	Eph    map[time.Time]Orbit // Time stamped heliocentric ephemeris.
}

// String implements the Stringer interface.
func (c *CelestialObject) String() string {
	return fmt.Sprintf("[Object %s]", c.Name)
}

// Equals returns whether the provided celestial object is the same.
func (c *CelestialObject) Equals(b CelestialObject) bool {
	return c.Name == b.Name && c.Radius == b.Radius && c.a == b.a && c.μ == b.μ && c.SOI == b.SOI && c.J2 == b.J2
}

// HelioOrbit returns the heliocentric position and velocity of this planet at a given time in equatorial coordinates.
// Note that the whole file is loaded. In fact, if we don't, then whoever is the first to call this function will
// set the Epoch at which the ephemeris are available, and that sucks.
func (c *CelestialObject) HelioOrbit(dt time.Time) ([]float64, []float64) {
	if c.Name == "Sun" {
		return []float64{0, 0, 0}, []float64{0, 0, 0}
	}
	if c.Eph == nil {
		// Load and parse the associated file.
		path := os.Getenv("HORIZON")
		if path == "" {
			panic("environment variable HORIZON not set")
		} else {
			fn := path + "/" + c.Name + ".csv"
			csvFile, err := os.Open(fn)
			if err != nil {
				panic(fmt.Errorf("could not load file %s", fn))
			}
			defer csvFile.Close()
			c.Eph = make(map[time.Time]Orbit)
			rdr := csv.NewReader(csvFile)
			for {
				record, err := rdr.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					panic(fmt.Errorf("error reading CSV file %s", err))
				}
				R := make([]float64, 3)
				V := make([]float64, 3)
				ephDT, err := time.ParseInLocation("2006-Jan-02 15:04:05.0000", strings.TrimSpace(record[1][5:]), time.UTC)
				if err != nil {
					panic(fmt.Errorf("could not parse UTC date %s", err))
				}
				for i := 0; i < 3; i++ {
					R[i], err = strconv.ParseFloat(strings.TrimSpace(record[i+2]), 64)
					if err != nil {
						panic(fmt.Errorf("could not parse float for R[%d] %s", i, err))
					}
					R[i] *= AU // Convert from AU to km
					V[i], err = strconv.ParseFloat(strings.TrimSpace(record[i+5]), 64)
					if err != nil {
						panic(fmt.Errorf("could not parse float for V[%d] %s", i, err))
					}
					V[i] *= AU / (3600 * 24) // Convert from AU/day to km/s
				}
				// Convert to equatorial.
				R = MxV33(R1(Deg2rad(-c.tilt)), R)
				V = MxV33(R1(Deg2rad(-c.tilt)), V)
				c.Eph[ephDT] = Orbit{R, V, Sun}
			}
		}
	}
	approxDT := dt.Round(time.Duration(1) * time.Minute)
	if o, exists := c.Eph[approxDT]; !exists {
		panic(fmt.Errorf("could not find date %s in %s ephemeris", approxDT, c.Name))
	} else {
		return o.R, o.V
	}
}

/* Definitions */

// Sun is our closest star.
var Sun = CelestialObject{"Sun", 695700, -1, 1.32712440018 * 1e11, 0.0, -1, -1, nil}

// Earth is home.
var Earth = CelestialObject{"Earth", 6378.1363, 149598023, 3.986004415 * 1e5, 23.4, 924645.0, 0.0010826269, nil}

// Mars is the vacation place.
var Mars = CelestialObject{"Mars", 3397.2, 227939186, 4.305 * 1e4, 25.19, 576000, 0.001964, nil}
