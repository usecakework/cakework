package cli

import (
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"
	"strings"

	flyTypes "github.com/usecakework/cakework/lib/fly"
	"github.com/usecakework/cakework/lib/shell"
)

type Fly struct {
	BinaryPath string // path to where the Fly CLI is installed locally
	AccessToken string
	Org string
}

func New(binaryPath string, accessToken string, org string) *Fly {
	fly := &Fly{
		BinaryPath: binaryPath,
		AccessToken: accessToken,
		Org: org,
	}

	return fly
}

func (fly *Fly) CreateApp(appName string, directory string) (string, error) {
	cmd := exec.Command(fly.BinaryPath, "apps", "create", appName, "--org", fly.Org, "-t", fly.AccessToken)
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

// TODO make it so that we don't allocate a new ipv4 for every deploy
func (fly *Fly) AllocateIpv4(appName string, directory string) (string, error) {
	cmd := exec.Command(fly.BinaryPath, "ips", "allocate-v4", "--app", appName, "-t", fly.AccessToken)
	return shell.RunCmd(cmd, directory)
}

// spins up a new machine
// this is just to trigger a deployment on fly's backend so that they build an image in their repo for us
func (fly *Fly) NewMachine(appName string, directory string) (string, error) {
		// since name not specified, fly will create a new name automatically
	// pick the smallest instance type
	config := flyTypes.MachineConfig {
		Config: flyTypes.Config{
			Guest: flyTypes.Guest{
				CPUKind:  "shared",
				CPUs:     1,
				MemoryMB: 256,
			},
			Restart : flyTypes.Restart {
				Policy: "no",
			},
		},
	}

	bytes, err := json.Marshal(config)
    if err != nil {
        return "", err
    }

	configStr := string(bytes)

	cmd := exec.Command(fly.BinaryPath, "m", "run", ".", "-a", appName, "-t", fly.AccessToken, "-c", configStr)
	return shell.RunCmd(cmd, directory)
}

func (fly *Fly) GetMachineInfo(out string) (machineId string, state string, image string, err error) {
	if strings.Contains(out, "Success") {
		machineId = findWithRegex(`Machine\sID:\s([^)]+)`, out)
		state = findWithRegex(`State:\s([^)]+)`, out)
		image = findWithRegex(`Image:\s([^)]+)`, out)
		
		if machineId == "" || state == "" || image == "" {
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

func SanitizeUserId(userId string) string {
	return strings.Replace(strings.ToLower(userId), "_", "-", -1)
}

func SanitizeAppName(app string) string {
	return strings.Replace(strings.ToLower(app), "_", "-", -1)
}

func SanitizeProjectName(project string) string {
	return strings.Replace(strings.ToLower(project), "_", "-", -1)
}

func SanitizeTaskName(task string) string {
	return strings.Replace(strings.ToLower(task), "_", "-", -1)
}

// should be sanitized for fly
func GetFlyAppName(userId string, appName string, taskName string) string {
	return SanitizeUserId(userId) + "-" + SanitizeAppName(appName) + "-" + SanitizeTaskName(taskName)
}