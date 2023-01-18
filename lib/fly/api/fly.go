package api

import (
	"errors"
	"fmt"
	"net/url"
	"path"

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

type Request struct {
	Name   string `json:"name"`
	Config Config `json:"config,omitempty"`
}

type Config struct {
	Image string `json:"image,omitempty"`
	Guest Guest  `json:"guest,omitempty"`
}

type Guest struct {
	CPUKind  string `json:"cpu_kind,omitempty"`
	CPUs     int    `json:"cpus,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
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
func (fly *Fly) NewMachine(flyApp string, name string, image string, cpus int, memoryMB int) error {
	// make a post request to the internal fly api endpoint
	url, _ := fly.MachineUrl(flyApp)

	fmt.Println("Calling: " + url + " to deploy new machine")

	req := Request{
		Name: flyApp,
		Config: Config{
			Image: image,
			Guest: Guest{
				CPUKind:  "shared", // Q: support dedicated?
				CPUs:     cpus,
				MemoryMB: memoryMB,
			},
		},
	}


	fmt.Println(req)

	data, res, err := http.Call(url, "POST", req, fly.CredentialsProvider)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		fmt.Println(res)
		fmt.Println(data)
		return errors.New("Failed to create new Fly machine")
	}
	return nil
}

func (fly *Fly) MachineUrl(flyApp string) (string, error) {
	u, err := url.Parse(fly.Endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "v1/apps", flyApp, "machines")
	return u.String(), nil
}
