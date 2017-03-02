package smd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	cfgLoaded = false
	config    = _smdconfig{}
)

// _smdconfig is a "hidden" struct, just use `smdConfig`
type _smdconfig struct {
	VSOP87, SPICE bool
	VSOP87Dir     string
	SPICEDir      string
	outputDir     string
}

func (c _smdconfig) ChgFrame(toFrame, fromFrame string, epoch time.Time, state []float64) ([]float64, []float64) {
	conf := smdConfig()
	stateStr := ""
	for _, val := range state {
		stateStr += fmt.Sprintf("%f,", val)
	}
	stateStr = fmt.Sprintf("[%s]", stateStr[:len(stateStr)-1]) // Trim the last comma
	cmd := exec.Command("python3", conf.SPICEDir+"/chgframe.py", "-t", toFrame, "-f", fromFrame, "-e", epoch.Format(time.ANSIC), "-s", stateStr)
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running chgframe: %s ", err)
		os.Exit(1)
	}
	return stateFromString(cmdOut)
}

func (c _smdconfig) HelioState(planet string, epoch time.Time) ([]float64, []float64) {
	conf := smdConfig()
	cmd := exec.Command("python3", conf.SPICEDir+"/heliostate.py", "-p", planet, "-e", epoch.Format(time.ANSIC))
	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running chgframe: %s ", err)
		os.Exit(1)
	}
	return stateFromString(cmdOut)
}

func stateFromString(cmdOut []byte) ([]float64, []float64) {
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
	return R, V
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
	vsop87Enabled := viper.GetBool("VSOP87.enabled")
	vsop87Dir := viper.GetString("VSOP87.directory")
	outputDir := viper.GetString("general.output_path")

	if vsop87Enabled && spiceEnabled {
		panic("both VSOP87 and SPICE are enabled, please make up your mind (SPICE is more precise)")
	}
	cfgLoaded = true
	return _smdconfig{VSOP87: vsop87Enabled, VSOP87Dir: vsop87Dir, SPICE: spiceEnabled, SPICEDir: spiceDir, outputDir: outputDir}
}
