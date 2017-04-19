package smd

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/soniakeys/meeus/julian"
	"github.com/spf13/viper"
)

var (
	cfgLoaded     = false
	config        = _smdconfig{}
	loadedCSVName = ""
	loadedCSVdata = make(map[string]map[time.Time]planetstate)
	spiceCSVMutex = &sync.Mutex{}
)

type planetstate struct {
	R, V []float64
}

// _smdconfig is a "hidden" struct, just use `smdConfig`
type _smdconfig struct {
	SPICEDir   string
	HorizonDir string
	outputDir  string
	spiceTrunc time.Duration
	spiceCSV   bool
	meeus      bool
	testExport bool
}

func (c _smdconfig) String() string {
	if c.spiceCSV {
		return fmt.Sprintf("[smd:config] SPICE: CSV - %s", c.HorizonDir)
	}
	return fmt.Sprintf("[smd:config] SPICE: SpiceyPy - %s", c.SPICEDir)
}

func (c _smdconfig) ChgFrame(toFrame, fromFrame string, epoch time.Time, state []float64) planetstate {
	conf := smdConfig()
	stateStr := ""
	for _, val := range state {
		stateStr += fmt.Sprintf("%f,", val)
	}
	stateStr = fmt.Sprintf("[%s]", stateStr[:len(stateStr)-1]) // Trim the last comma
	cmd := exec.Command("python", conf.SPICEDir+"/chgframe.py", "-t", toFrame, "-f", fromFrame, "-e", epoch.Format(time.ANSIC), "-s", stateStr)
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command attempted:\npython %s/chgframe.py -t %s -f %s -e \"%s\" -s %s\n", conf.SPICEDir, toFrame, fromFrame, epoch.Format(time.ANSIC), stateStr)
		panic(fmt.Errorf("error running chgframe: %s \ncheck that you are in the Python virtual environment", err))
	}
	return stateFromString(cmdOut)
}

func (c _smdconfig) HelioState(planet string, epoch time.Time) planetstate {
	epoch = epoch.UTC()
	conf := smdConfig()
	if conf.meeus {
		if planet != "Earth" {
			panic("Meeus only supports Earth ephemerides")
		}
		t := (julian.TimeToJD(epoch) - 2451545.0) / 36525
		tVec := []float64{1, t, t * t, t * t * t}
		/* Earth coeffs */
		L := []float64{100.466449, 35999.3728519, -0.00000568, 0.0}
		a := []float64{1.000001018, 0.0, 0.0, 0.0}
		eVec := []float64{0.01670862, -0.000042037, -0.0000001236, 0.00000000004}
		i := []float64{0.0, 0.0130546, -0.00000931, -0.000000034}
		W := []float64{174.873174, -0.2410908, 0.00004067, -0.000001327}
		P := []float64{102.937348, 0.3225557, 0.00015026, 0.000000478}
		valL := dot(L, tVec) * deg2rad
		valSMA := dot(a, tVec) * AU
		e := dot(eVec, tVec)
		valInc := dot(i, tVec) * deg2rad
		valW := dot(W, tVec) * deg2rad
		valP := dot(P, tVec) * deg2rad
		w := valP - valW
		M := valL - valP
		Ccen := (2*e-math.Pow(e, 3)/4+5./96*math.Pow(e, 5))*math.Sin(M) + (5./4*math.Pow(e, 2)-11./24*math.Pow(e, 4))*math.Sin(2*M) + (13./12*math.Pow(e, 3)-43./64*math.Pow(e, 5))*math.Sin(3*M) + 103./96*math.Pow(e, 4)*math.Sin(4*M) + 1097./960*math.Pow(e, 5)*math.Sin(5*M)
		nu := M + Ccen
		R, V := NewOrbitFromOE(valSMA, e, valInc, valW, w, nu, Sun).RV()
		return planetstate{R, V}
	} else if conf.spiceCSV {
		spiceCSVMutex.Lock() // Data race if a given thread tries to read from the map while it's loading and the data isn't fully loaded yet.
		ephemeride := fmt.Sprintf("%s-%04d", planet, epoch.Year())
		if _, found := loadedCSVdata[ephemeride]; !found {
			loadedCSVdata[ephemeride] = make(map[time.Time]planetstate)
			// Let's load a new file.
			loadingProfileDT := time.Now()
			file, err := os.Open(fmt.Sprintf("%s/%s.csv", conf.HorizonDir, ephemeride))
			if err != nil {
				log.Printf("%s\nGenerating it now...", err)
				// Generate it.
				cmd := exec.Command("python", conf.SPICEDir+"/horizon.py", "-p", planet, "-y", fmt.Sprintf("%d", epoch.Year()), "-r", "1m")
				_, err := cmd.Output()
				if err != nil {
					panic(fmt.Errorf("error running horizon: %s \ncheck that you are in the Python virtual environment", err))
				}
				log.Println("[OK]")
				// Load the file again and totally fail if issue.
				file, err = os.Open(fmt.Sprintf("%s/%s.csv", conf.HorizonDir, ephemeride))
				if err != nil {
					log.Fatalf("could not open file after generation: %s", err)
				}
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				entries := strings.Split(scanner.Text(), ",")
				// Parse the data.
				dt, err := time.Parse("2006-1-2T15:4:5", entries[1])
				if err != nil {
					panic("could not parse date time")
				}
				// Drop the string of the date
				R := make([]float64, 3)
				V := make([]float64, 3)
				for i := 0; i < 3; i++ {
					tR, err := strconv.ParseFloat(strings.TrimSpace(entries[i+2]), 64)
					if err != nil {
						log.Fatalf("[smd:error] could not parse position when reading %s: `%s`", ephemeride, strings.TrimSpace(entries[i+2]))
					}
					tV, err := strconv.ParseFloat(strings.TrimSpace(entries[i+5]), 64)
					if err != nil {
						log.Fatalf("[smd:error] could not parse velocity when reading %s: `%s`", ephemeride, strings.TrimSpace(entries[i+5]))
					}
					R[i] = tR
					V[i] = tV
				}
				loadedCSVdata[ephemeride][dt] = planetstate{R, V}
			}

			if err := scanner.Err(); err != nil {
				log.Fatalf("[smd:error] %s when loading %s", err, ephemeride)
			}
			fmt.Printf("[smd:info] %s loaded in %s\n", ephemeride, time.Now().Sub(loadingProfileDT))
		}
		// And now let's find the state.
		state, found := loadedCSVdata[ephemeride][epoch.Truncate(conf.spiceTrunc)]
		if !found {
			log.Fatalf("state at date %s (%f) not found in %s.csv, try regenerating the set", epoch.Truncate(conf.spiceTrunc), julian.TimeToJD(epoch.Truncate(conf.spiceTrunc)), ephemeride)
		}
		spiceCSVMutex.Unlock()
		return state
	}
	cmd := exec.Command("python", conf.SPICEDir+"/heliostate.py", "-p", planet, "-e", epoch.Format(time.ANSIC))
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command attempted:\npython %s/heliostate.py -p %s -e \"%s\"\n", conf.SPICEDir, planet, epoch.Format(time.ANSIC))
		panic(fmt.Errorf("error running heliostate: %s \ncheck that you are in the Python virtual environment", err))
	}
	return stateFromString(cmdOut)

}

func stateFromString(cmdOut []byte) planetstate {
	newStateStr := strings.TrimSpace(string(cmdOut))
	newStateStr = newStateStr[1 : len(newStateStr)-1]
	components := strings.Split(newStateStr, ",")
	var R = make([]float64, 3)
	var V = make([]float64, 3)
	for i := 0; i < 6; i++ {
		fl, err := strconv.ParseFloat(strings.TrimSpace(components[i]), 64)
		if err != nil {
			panic(err)
		}
		if i < 3 {
			R[i] = fl
		} else {
			V[i-3] = fl
		}
	}
	return planetstate{R, V}
}

// getSMDConfig returns the smd configuration.
func smdConfig() _smdconfig {
	if cfgLoaded {
		return config
	}
	// Load the configuration file
	confPath := os.Getenv("SMD_CONFIG")
	if confPath == "" {
		panic("environment variable `SMD_CONFIG` is missing or empty")
	}
	viper.SetConfigName("conf")
	viper.AddConfigPath(confPath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("%s/conf.toml not found", confPath))
	}

	spiceDir := viper.GetString("SPICE.directory")
	spiceCSV := viper.GetBool("SPICE.horizonCSV")
	spiceCSVDir := viper.GetString("SPICE.HorizonDir")
	spiceTruncationStr := viper.GetString("SPICE.truncation")
	var spiceTruncation time.Duration
	var derr error
	if spiceTruncation, derr = time.ParseDuration(spiceTruncationStr); derr != nil && spiceCSV {
		fmt.Println("[ERROR] Could not parse spice truncation, using 1 second")
		spiceTruncation = time.Minute // Default value
	}
	outputDir := viper.GetString("general.output_path")
	testExport := viper.GetBool("general.test_export")
	meeus := viper.GetBool("Meeus.enabled")
	if meeus {
		fmt.Println("\nWARNING: Meeus enabled, supersedes SPICE")
	}

	cfgLoaded = true
	config = _smdconfig{SPICEDir: spiceDir, spiceTrunc: spiceTruncation, spiceCSV: spiceCSV, HorizonDir: spiceCSVDir, outputDir: outputDir, testExport: testExport, meeus: meeus}
	return config
}
