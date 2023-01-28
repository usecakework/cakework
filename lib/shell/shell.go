package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

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

func RunCmdLive(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	// merge stderr into stdout
	cmd.Stderr = cmd.Stdout

	defer stdout.Close()
	buf := bufio.NewReader(stdout)

	if err := cmd.Start(); err != nil {
		return err
	}

	for {
		str, err := buf.ReadString('\n')
		// TODO look for specific eof err
		if err != nil {
			break
		}
		fmt.Print(str)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				log.Debug("Exit Status: ", status.ExitStatus())
				return err
			}
		}
		return err
	}

	return nil
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
