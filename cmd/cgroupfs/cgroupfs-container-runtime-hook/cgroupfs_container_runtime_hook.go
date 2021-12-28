package main

import "C"
import (
	"encoding/json"
	"flag"
	"fmt"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	//"github.com/zimmski/osutil"

	// "github.com/openkhal/container-runtime-box/pkg/nsenter"

	"k8s.io/klog"
	"log"
	"os"
	"runtime"
	"time"
)

///*
//extern void enter_namespace(void);
//*/
//import "C"

var (
	logFileName = "/var/log/cgroupfs-container-runtime-hook.log"
)

const (
	Cgroupfs     = "/cgroupfs"
	CgroupfsPath = "/cgroupfs/%s"
	ProcPath     = "proc/%s"
	SysPath      = "/sys/%s"
)

var BindResources = []string{"cpuinfo", "stat", "meminfo", "loadavg", "uptime", "vmstat"}

func setLog() {
	logfile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logfile)
	log.SetPrefix(time.Now().Format("2006-01-02 15:04:05") + "[" + fmt.Sprintf("%d", time.Now().UnixNano()) + "]" + " [Prestart] ")
}

var (
	debugflag  = flag.Bool("debug", false, "enable debug output")
	configflag = flag.String("config", "", "configuration file")

	defaultPATH = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}
)

func exit() {
	if err := recover(); err != nil {
		if _, ok := err.(runtime.Error); ok {
			klog.Errorln(err)
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func getDstPath(containerPath string, containerSpec spec.Spec, bundle string) string {
	var AbsDst string
	if path.IsAbs(containerSpec.Root.Path) {
		AbsDst = path.Join(containerSpec.Root.Path, containerPath)
	} else {
		AbsDst = path.Join(path.Join(bundle, containerSpec.Root.Path), containerPath)
	}
	return AbsDst
}

func getEnvFromSpec(envName string, envs []string) string {
	envName = envName + "="

	for _, env := range envs {
		if strings.HasPrefix(env, envName) {
			idx := strings.Index(env, "=")
			if idx != -1 {
				return env[idx+1:]
			}
		}
	}

	return ""
}

func doBindMount(source string, containerSpec spec.Spec, bundle string, pid int64) {
	absSrc := fmt.Sprintf(CgroupfsPath, source)
	absDst := getDstPath(source, containerSpec, bundle)

	log.Println("absSrc", absSrc)
	log.Println("absDst", absDst)

	mountCmd := exec.Command("/usr/bin/mount_cgroup", fmt.Sprintf("%d", pid), absSrc, absDst)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to execute mount, output:%s err:%v\n", string(output), err)
		return

	}
	log.Printf("mount output: %s\n", output)
}

func Mounted(mountPoint string) (bool, error) {
	mntPoint, err := os.Stat(mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	parent, err := os.Stat(filepath.Join(mountPoint, ".."))
	if err != nil {
		return false, err
	}
	mntPointSt := mntPoint.Sys().(*syscall.Stat_t)
	parentSt := parent.Sys().(*syscall.Stat_t)
	return mntPointSt.Dev != parentSt.Dev, nil
}

func doPrestart() {
	defer exit()
	log.Printf("Copy stdin to prestart hook\n")
	hookSpecBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("Fail to read from stdin")
		return
	}
	log.Printf("hookSpecBuf: %s\n", hookSpecBuf)

	containerSpec := spec.Spec{}
	hookSpec := make(map[string]interface{})
	decoder := json.NewDecoder(strings.NewReader(string(hookSpecBuf)))
	decoder.UseNumber()
	decoder.Decode(&hookSpec)
	log.Printf("data: %#+v\n", hookSpec)

	bundleElem, exists := hookSpec["bundle"]
	if !exists {
		log.Printf("Did not find bundle in hookSpec\n")
		return
	}
	bundle, ok := bundleElem.(string)
	if !ok {
		log.Printf("Bundle is not a string")
		return
	}
	log.Printf("Get bundle: %s", string(bundle))

	specFile := path.Join(bundle, "config.json")
	log.Printf("Container spec file path:%s\n", specFile)

	containerSpec, err = loadSpec(specFile)
	if err != nil {
		log.Printf("Fail to get container spec %v\n", err)
		return
	}
	pidElem, exists := hookSpec["pid"]
	if !exists {
		log.Printf("No pid exists in hook spec\n")
		return
	}
	pidJSON, ok := pidElem.(json.Number)
	if !ok {
		log.Printf("No pid exists in hook spec\n")
		return
	}
	pid, _ := pidJSON.Int64()
	//bindNeed := getEnvFromSpec("BIND_MOUNT", containerSpec.Process.Env)
	//log.Printf("containerSpec.Process.Env",containerSpec.Process.Env)
	//if bindNeed == "" {
	//	log.Printf("Need't bind mount")
	//	return
	//} else {
	if mounted, err := Mounted(Cgroupfs); err != nil {
		log.Printf("Fail to mount cgroupfs because mounted is %s",err)
		return
	} else if mounted {
		log.Printf("%s is already mounted", Cgroupfs)
	}else {
		if err := os.Mkdir(Cgroupfs, 0755); err != nil && !os.IsExist(err) {
			log.Printf("Fail to mount cgroupfs because mkdir cgroupfs is %s",err)
			return
		}
		if err := syscall.Mount("cgroupfs", Cgroupfs, "cgroupfs", 0, "ro"); err != nil {
			log.Printf("Fail to mount cgroupfs because of %s",err)
			return
		}
		log.Println("Successfully mount for cgroupfs")

	}


	//bindSet := make(map[string]struct{})
	//for _, resource := range BindResources {
	//	bindSet[resource] = struct{}{}
	//}
	//bindMountArray := strings.Split(bindNeed, ",")
	//log.Println("bindSet",bindSet)

	//log.Println("bindMountArray",bindMountArray)
	for _, resource := range BindResources {
		log.Println("bindMount--------->",resource)
		doBindMount(fmt.Sprintf(ProcPath, resource), containerSpec, bundle, pid)
	}

	//doBindMount(fmt.Sprintf(SysPath,"devices/system/cpu/"), containerSpec, bundle, pid)





}

func loadSpec(path string) (spec spec.Spec, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&spec); err != nil {
		return
	}
	if spec.Version == "" {
		err = fmt.Errorf("version is empty in OCI spec")
		return
	}
	if spec.Process == nil {
		err = fmt.Errorf("process is empty in OCI spec")
		return
	}
	if spec.Root == nil {
		err = fmt.Errorf("root is empty in OCI spec")
		return
	}

	return
}

//func init() {
//	runtime.GOMAXPROCS(1)
//	runtime.LockOSThread()
//	setLog()
//
//	hookSpecBuf, err := ioutil.ReadAll(os.Stdin)
//	if err != nil {
//		log.Panicln(err)
//		return
//	}
//	decoder := json.NewDecoder(strings.NewReader(string(hookSpecBuf)))
//	decoder.UseNumber()
//	decoder.Decode(&hookSpec)
//	pidElem, exists := hookSpec["pid"]
//	if !exists {
//		log.Printf("No pid exists in hook spec\n")
//		return
//	}
//	pidJSON, ok := pidElem.(json.Number)
//	if !ok {
//		log.Printf("No pid exists in hook spec\n")
//		return
//	}
//	pid, _ = pidJSON.Int64()
//	os.Setenv("mydocker_pid", strconv.FormatInt(pid, 10))
//	//output, _ := osutil.CaptureWithCGo(enterNamespace)
//	//log.Println(string(output))
//}

//func enterNamespace() {
//	C.enter_namespace()
//}

func main() {
	flag.Parse()
	args := flag.Args()
	setLog()

	switch args[0] {
	case "prestart":
		doPrestart()
		os.Exit(0)
	default:
		os.Exit(2)
	}
}
