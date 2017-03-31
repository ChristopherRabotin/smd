package smd

import (
	"bufio"
	"fmt"
	"log"
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
	VSOP87, SPICE bool
	VSOP87Dir     string
	SPICEDir      string
	HorizonDir    string
	outputDir     string
	spiceTrunc    time.Duration
	spiceCSV      bool
	testExport    bool
}

func (c _smdconfig) ChgFrame(toFrame, fromFrame string, epoch time.Time, state []float64) planetstate {
	conf := smdConfig()
	stateStr := ""
	for _, val := range state {
		stateStr += fmt.Sprintf("%f,", val)
	}
	stateStr = fmt.Sprintf("[%s]", stateStr[:len(stateStr)-1]) // Trim the last comma
	cmd := exec.Command("python3", conf.SPICEDir+"/chgframe.py", "-t", toFrame, "-f", fromFrame, "-e", epoch.Format(time.ANSIC), "-s", stateStr)
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command attempted:\npython3 %s/chgframe.py -t %s -f %s -e \"%s\" -s %s\n", conf.SPICEDir, toFrame, fromFrame, epoch.Format(time.ANSIC), stateStr)
		panic(fmt.Errorf("error running chgframe: %s \ncheck that you are in the Python virtual environment", err))
	}
	return stateFromString(cmdOut)
}

func (c _smdconfig) HelioState(planet string, epoch time.Time) planetstate {
	epoch = epoch.UTC()
	conf := smdConfig()
	if conf.spiceCSV {
		spiceCSVMutex.Lock() // Data race if a given thread tries to read from the map while it's loading and the data isn't fully loaded yet.
		ephemeride := fmt.Sprintf("%s-%04d", planet, epoch.Year())
		if _, found := loadedCSVdata[ephemeride]; !found {
			loadedCSVdata[ephemeride] = make(map[time.Time]planetstate)
			// Let's load a new file.
			loadingProfileDT := time.Now()
			file, err := os.Open(fmt.Sprintf("%s/%s.csv", conf.HorizonDir, ephemeride))
			if err != nil {
				log.Fatal(err)
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
						panic("could not parse position")
					}
					tV, err := strconv.ParseFloat(strings.TrimSpace(entries[i+5]), 64)
					if err != nil {
						panic("could not parse velocity")
					}
					R[i] = tR
					V[i] = tV
				}
				loadedCSVdata[ephemeride][dt] = planetstate{R, V}
			}

			if err := scanner.Err(); err != nil {
				panic(err)
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
	cmd := exec.Command("python3", conf.SPICEDir+"/heliostate.py", "-p", planet, "-e", epoch.Format(time.ANSIC))
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command attempted:\npython3 %s/heliostate.py -p %s -e \"%s\"\n", conf.SPICEDir, planet, epoch.Format(time.ANSIC))
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

	spiceEnabled := viper.GetBool("SPICE.enabled")
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
	vsop87Enabled := viper.GetBool("VSOP87.enabled")
	vsop87Dir := viper.GetString("VSOP87.directory")
	outputDir := viper.GetString("general.output_path")
	testExport := viper.GetBool("general.test_export")

	if vsop87Enabled && spiceEnabled {
		panic("both VSOP87 and SPICE are enabled, please make up your mind (SPICE is more precise)")
	}
	cfgLoaded = true
	config = _smdconfig{VSOP87: vsop87Enabled, VSOP87Dir: vsop87Dir, SPICE: spiceEnabled, SPICEDir: spiceDir, spiceTrunc: spiceTruncation, spiceCSV: spiceCSV, HorizonDir: spiceCSVDir, outputDir: outputDir, testExport: testExport}
	return config
}
