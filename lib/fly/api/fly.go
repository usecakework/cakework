package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/usecakework/cakework/lib/auth"
	"github.com/usecakework/cakework/lib/http"
)

/**
VM options
fly platform vm-sizes
NAME            	CPU CORES	MEMORY
shared-cpu-1x   	1        	256 MB
dedicated-cpu-1x	1        	2 GB
dedicated-cpu-2x	2        	4 GB
dedicated-cpu-4x	4        	8 GB
dedicated-cpu-8x	8        	16 GB

Ex: fly scale vm dedicated-cpu-1x --memory 4096 (for apps not machine)
**/

type Fly struct {
	Org                 string
	Endpoint            string
	CredentialsProvider auth.BearerStringCredentialsProvider
}

type MachineConfig struct {
	Name   string `json:"name,omitempty"`
	Config Config `json:"config,omitempty"`
	MachineId string `json:"id,omitempty"`
}

type Restart struct {
	Policy string `json:"policy,omitempty"`
}

type Config struct {
	Image string `json:"image,omitempty"`
	Guest Guest  `json:"guest,omitempty"`
	Restart Restart `json:"restart,omitempty"`
}

type Guest struct {
	CPUKind  string `json:"cpu_kind,omitempty"`
	CPUs     int    `json:"cpus,omitempty"`
	Memory int    `json:"memory_mb,omitempty"`
}

func New(org string, endpoint string, credentialsProvider auth.BearerStringCredentialsProvider) *Fly {
	fly := &Fly{
		Org:                 org,
		Endpoint:            endpoint,
		CredentialsProvider: credentialsProvider,
	}

	return fly
}

// Q: should this return machine info?
func (fly *Fly) NewMachine(flyApp string, name string, image string, cpus int, memory int) (MachineConfig, error) {
	var config MachineConfig
	// make a post request to the internal fly api endpoint
	url, _ := fly.AppUrl(flyApp)

	fmt.Println("Calling: " + url + " to deploy new machine")

	req := MachineConfig {
		Name: name,
		Config: Config{
			Image: image,
			Guest: Guest{
				CPUKind:  "shared", // Q: support dedicated?
				CPUs:     cpus,
				Memory: memory,
			},
			Restart : Restart {
				Policy: "no",
			},
		},
	}

	// TODO delete
	// fmt.Printf("%+v\n", req)

	res, err := http.CallV2(url, "POST", req, fly.CredentialsProvider)
	if err != nil {
		return config, err
	}
	if res.StatusCode != 200 {
		fmt.Println(res)
		fmt.Println(res.StatusCode)
		return config, errors.New("Failed to create new Fly machine")
	} else {
		var config MachineConfig
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return config, err
		}

		json.Unmarshal(body, &config)
		log.Printf("Successfully deployed new machine! %+v\n", config)
		return config, nil
	}
}

func (fly *Fly) Wait(flyApp string, machineId string, state string) error {
	url, err := fly.MachineUrl(flyApp, machineId)
	if err != nil {
		return err
	}

	
	// hardcoded
	res, err := http.CallV2(url + "?state=started&timeout=60", "GET", nil, fly.CredentialsProvider)
	if err != nil {
		log.Error("Error while waiting for Fly Machine to reach desired state")
		return err
	}

	defer res.Body.Close()
	
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 {
		log.Info("Fly machine reached " + state + " state")
		return nil
	} else {
		log.Error(res)
		return errors.New("Machine failed to reach desired state (Fly returned non-200 code)")
	}
}

func (fly *Fly) AppUrl(flyApp string) (string, error) {
	u, err := url.Parse(fly.Endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "v1/apps", flyApp, "machines")
	return u.String(), nil
}

func (fly *Fly) MachineUrl(flyApp string, machineId string) (string, error) {
	u, err := url.Parse(fly.Endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "v1/apps", flyApp, "machines", machineId, "wait")
	return u.String(), nil}
