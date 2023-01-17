package fly

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"

	"github.com/usecakework/cakework/shell"
)

type Fly struct {
	binaryPath string // path to where the Fly CLI is installed locally
	accessToken string
	org string
}

func New(binaryPath string, accessToken string, org string) *Fly {
	fly := &Fly{
		binaryPath: binaryPath,
		accessToken: accessToken,
		org: org,
	}

	return fly
}

func (fly *Fly) CreateApp(appName string, directory string) (string, error) {
	cmd := exec.Command(fly.binaryPath, "apps", "create", appName, "--org", fly.org, "-t", fly.accessToken)
	out, err := shell.RunCmdSilent(cmd, directory) // silent so that we don't print out an error if name has already been taken
	if err != nil {
		if strings.Contains(out, "Name has already been taken") {
			return out, nil
		} else {
			return out, err
		}
	}
	return out, nil
}

func (fly *Fly) AllocateIpv4(appName string, directory string) (string, error) {
	cmd := exec.Command(fly.binaryPath, "ips", "allocate-v4", "--app", appName, "-t", fly.accessToken)
	return shell.RunCmd(cmd, directory)
}

// spins up a new machine
func (fly *Fly) DeployMachine(appName string, directory string) (string, error) {
	cmd := exec.Command(fly.binaryPath, "m", "run", ".", "-a", appName, "-t", fly.accessToken)
	return shell.RunCmd(cmd, directory)
}

func (fly *Fly) GetMachineInfo(out string) (machineId string, instanceId string, state string, image string, err error) {
	if strings.Contains(out, "Success") {
		machineId = findWithRegex(`Machine\sID:\s([^)]+)`, out)
		instanceId = findWithRegex(`Instance\sID:\s([^)]+)`, out)
		state = findWithRegex(`State:\s([^)]+)`, out)
		image = findWithRegex(`Image:\s([^)]+)`, out)
		
		if machineId == "" || instanceId == "" || state == "" || image == "" {
			err = errors.New("Could not get Fly machine info")	
		}
	} else {
		err = errors.New("Could not get Fly machine info")
	}
	return
}

// func updateMachine(machineId string)

func findWithRegex(rgxString string, s string) string {
	rgx := regexp.MustCompile(rgxString)
	sList := strings.Split(s, "\n")
	for _, line := range sList {
		rs := rgx.FindAllStringSubmatch(line, -1)
		if len(rs) > 0 {
			for _, i := range rs {
				return i[1]
			}
		}
	}
	return ""
}