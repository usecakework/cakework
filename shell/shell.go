package shell

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// do not print out, err, and stderr if encounter error
func RunCmdSilent(cmd *exec.Cmd, directory string) (string, error) {
	log.Debug("executing command: " + strings.Join(cmd.Args, " "))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if directory != "" {
		cmd.Dir = directory
	}
	err := cmd.Run()
	allOutput := fmt.Sprint(err) + ": " + stderr.String() + ": " + out.String()

	if err != nil {
		return allOutput, err
	}
	return allOutput, nil
}

// execute a command and print out outputs
// TODO take a parameter to execute command in a particular directory
// TODO make it so that directory is optional
func RunCmd(cmd *exec.Cmd, directory string) (string, error) {
	log.Debug("executing command: " + strings.Join(cmd.Args, " "))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if directory != "" {
		cmd.Dir = directory
	}
	err := cmd.Run()
	allOutput := fmt.Sprint(err) + ": " + stderr.String() + ": " + out.String()

	if err != nil {
		fmt.Println(out.String())
		fmt.Println(err)
		fmt.Println(stderr.String())
		// since sometimes errors are printed to stdout instead of stderr, print out stdout as well
		return allOutput, err
	}
	return allOutput, nil
}