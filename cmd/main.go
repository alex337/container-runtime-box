package main

import (
	"fmt"
	"github.com/openkhal/container-runtime-box/pkg/oci"
	"github.com/openkhal/container-runtime-box/pkg/runtimes"
	"github.com/openkhal/container-runtime-box/pkg/utils"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

const (
	configOverride = "XDG_CONFIG_HOME"
	configFilePath = "container-runtime-box/config.toml"
)

var (
	configDir = "/etc/"
)

var logger = utils.NewLogger()

func main() {
	err := run(os.Args)
	if err != nil {
		logger.Errorf("error running %v: %v", os.Args, err)
		os.Exit(1)
	}
}

// run is an entry point that allows for idiomatic handling of errors
// when calling from the main function.
func run(argv []string) (err error) {
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}
	err = logger.LogToFile(cfg.debugFilePath)
	if err != nil {
		return fmt.Errorf("error opening debug log file: %v", err)
	}
	defer func() {
		// We capture and log a returning error before closing the log file.
		if err != nil {
			logger.Errorf("Error running %v: %v", argv, err)
		}
		logger.CloseFile()
	}()

	runtimeFactory := runtimes.NewRuntimeFactory(logger.Logger)
	runtime, err := runtimeFactory.NewRuncRuntime()
	if err != nil {
		return fmt.Errorf("error creating runtime: %v", err)
	}

	rs := []func(*log.Logger, oci.Runtime, oci.Spec) (oci.Runtime, error){runtimes.NewCgroupfsContainerRuntimeWithLogger, runtimes.NewNvidiaContainerRuntimeWithLogger}
	for _, r := range rs {
		runtime, err = runtimeFactory.BuildRuntime(runtime, r, argv)
		if err != nil {
			return fmt.Errorf("error creating runtime: %v", err)
		}
	}

	logger.Infof("Running %s\n", argv[0])
	logger.Infof("runtime %+v\n", runtime)
	return runtime.Exec(argv)
}

type config struct {
	debugFilePath string
}

// getConfig sets up the config struct. Values are read from a toml file
// or set via the environment.
func getConfig() (*config, error) {
	cfg := &config{}

	if XDGConfigDir := os.Getenv(configOverride); len(XDGConfigDir) != 0 {
		configDir = XDGConfigDir
	}

	configFilePath := path.Join(configDir, configFilePath)

	tomlContent, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	toml, err := toml.Load(string(tomlContent))
	if err != nil {
		return nil, err
	}

	cfg.debugFilePath = toml.GetDefault("debug", "/dev/null").(string)
	return cfg, nil
}
