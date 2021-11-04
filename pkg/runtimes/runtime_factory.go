package runtimes

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/openkhal/container-runtime-box/pkg/oci"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ociSpecFileName          = "config.json"
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
)

type RuntimeFactory struct {
	logger *log.Logger
}

func NewRuntimeFactory(logger *log.Logger) *RuntimeFactory {
	return &RuntimeFactory{
		logger: logger,
	}
}

func (rf *RuntimeFactory) BuildRuntime(runtime oci.Runtime, runtimeFunc func(*log.Logger, oci.Runtime, oci.Spec) (oci.Runtime, error), argv []string) (oci.Runtime, error) {
	ociSpec, err := rf.newOCISpec(argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	r, err := runtimeFunc(rf.logger, runtime, ociSpec)
	if err != nil {
		return nil, fmt.Errorf("error constructing NVIDIA Container Runtime: %v", err)
	}

	return r, nil
}

// newOCISpec constructs an OCI spec for the provided arguments
func (rf *RuntimeFactory) newOCISpec(argv []string) (oci.Spec, error) {
	bundlePath, err := rf.getBundlePath(argv)
	if err != nil {
		return nil, fmt.Errorf("error parsing command line arguments: %v", err)
	}

	ociSpecPath, err := rf.getOCISpecFilePath(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("error getting OCI specification file path: %v", err)
	}
	ociSpec := oci.NewSpecFromFile(ociSpecPath)

	return ociSpec, nil
}

func (rf *RuntimeFactory) NewRuncRuntime() (oci.Runtime, error) {
	runtimePath, err := rf.findRunc()
	if err != nil {
		return nil, fmt.Errorf("error locating runtime: %v", err)
	}

	runc, err := oci.NewSyscallExecRuntimeWithLogger(rf.logger, runtimePath)
	if err != nil {
		return nil, fmt.Errorf("error constructing runtime: %v", err)
	}

	return runc, nil
}

// getBundlePath checks the specified slice of strings (argv) for a 'bundle' flag as allowed by runc.
// The following are supported:
// --bundle{{SEP}}BUNDLE_PATH
// -bundle{{SEP}}BUNDLE_PATH
// -b{{SEP}}BUNDLE_PATH
// where {{SEP}} is either ' ' or '='
func (rf *RuntimeFactory) getBundlePath(argv []string) (string, error) {
	var bundlePath string

	for i := 0; i < len(argv); i++ {
		param := argv[i]

		parts := strings.SplitN(param, "=", 2)
		if !isBundleFlag(parts[0]) {
			continue
		}

		// The flag has the format --bundle=/path
		if len(parts) == 2 {
			bundlePath = parts[1]
			continue
		}

		// The flag has the format --bundle /path
		if i+1 < len(argv) {
			bundlePath = argv[i+1]
			i++
			continue
		}

		// --bundle / -b was the last element of argv
		return "", fmt.Errorf("bundle option requires an argument")
	}

	return bundlePath, nil
}

// findRunc locates runc in the path, returning the full path to the
// binary or an error.
func (rf *RuntimeFactory) findRunc() (string, error) {
	runtimeCandidates := []string{
		dockerRuncExecutableName,
		runcExecutableName,
	}

	return rf.findRuntime(runtimeCandidates)
}

func (rf *RuntimeFactory) findRuntime(runtimeCandidates []string) (string, error) {
	for _, candidate := range runtimeCandidates {
		rf.logger.Infof("Looking for runtime binary '%v'", candidate)
		runcPath, err := exec.LookPath(candidate)
		if err == nil {
			rf.logger.Infof("Found runtime binary '%v'", runcPath)
			return runcPath, nil
		}
		glog.V(5).Infof("Runtime binary '%v' not found: %v", candidate, err)
	}

	return "", fmt.Errorf("no runtime binary found from candidate list: %v", runtimeCandidates)
}

func isBundleFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}

	trimmed := strings.TrimLeft(arg, "-")
	return trimmed == "b" || trimmed == "bundle"
}

// getOCISpecFilePath returns the expected path to the OCI specification file for the given
// bundle directory. If the bundle directory is empty, only `config.json` is returned.
func (rf *RuntimeFactory) getOCISpecFilePath(bundleDir string) (string, error) {
	rf.logger.Infof("Using bundle directory: %v", bundleDir)

	OCISpecFilePath := filepath.Join(bundleDir, ociSpecFileName)

	rf.logger.Infof("Using OCI specification file path: %v", OCISpecFilePath)

	return OCISpecFilePath, nil
}
