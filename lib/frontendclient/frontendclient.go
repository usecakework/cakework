package frontendclient

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/usecakework/cakework/lib/auth"
	fly "github.com/usecakework/cakework/lib/fly/cli"
	"github.com/usecakework/cakework/lib/http"
	"github.com/usecakework/cakework/lib/types"
)

type Client struct {
	Url                 string
	CredentialsProvider auth.CredentialsProvider
}

func New(url string, credentialsProvider auth.CredentialsProvider) *Client {
	// func New(url string, accessToken string, refreshToken string, apiKey string) *Client {
	client := &Client{
		Url:                 url,
		CredentialsProvider: credentialsProvider,
	}

	return client
}

// TODO change all instances of appName to project
func (client *Client) CreateMachine(userId string, project string, task string, name string, machineId string, state string, image string) error {
	flyApp := fly.GetFlyAppName(userId, project, task)

	url := client.Url + "/create-machine"
	req := types.CreateMachineRequest{
		UserId:    userId,
		Project:   project,
		Task:      task,
		FlyApp:    flyApp,
		Name:      name,
		MachineId: machineId,
		State:     state,
		Image:     image,
	}

	_, res, err := http.Call(url, "POST", req, client.CredentialsProvider)
	if err != nil {
		return err
	}

	if res.StatusCode != 201 {
		// TODO pass up body string as well
		return errors.New("Failed to call Frontend to create machine. " + res.Status)
	}

	return nil
}

func (client *Client) GetUser(userId string) (*types.User, error) {
	url := client.Url + "/get-user"
	getUserRequest := types.GetUserRequest{
		UserId: userId,
	}

	body, res, err := http.Call(url, "GET", getUserRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		userId := body["id"].(string)
		return &types.User{Id: userId}, nil
	} else {
		return nil, errors.New("Error getting user details." + res.Status)
	}
}

func (client *Client) CreateUser(userId string) (*types.User, error) { // TODO change return type
	url := client.Url + "/create-user"
	createUserRequest := types.CreateUserRequest{
		UserId: userId,
	}

	body, res, err := http.Call(url, "POST", createUserRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		userId := body["id"].(string)
		return &types.User{Id: userId}, nil
	} else {
		return nil, errors.New("Error creating a new user." + res.Status)
	}
}

func (client *Client) CreateClientToken(userId string, name string) (*types.ClientToken, error) { // TODO change return type
	url := client.Url + "/create-client-token"
	createTokenReq := types.CreateTokenRequest{
		UserId: userId,
		Name:   name,
	}

	body, res, err := http.Call(url, "POST", createTokenReq, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 201 {
		token := body["token"].(string)
		return &types.ClientToken{Token: token}, nil
	} else {
		return nil, errors.New("Error creating client token" + res.Status)
	}
}

func (client *Client) GetRequestStatus(userId string, requestId string) (string, error) {
	url := client.Url + "/get-status"
	getStatusRequest := types.GetStatusRequest{
		UserId:    userId,
		RequestId: requestId,
	}

	body, res, err := http.Call(url, "GET", getStatusRequest, client.CredentialsProvider)
	if err != nil {
		return "", err
	}

	if res.StatusCode == 200 {
		status := body["status"].(string)
		return status, nil
	} else if res.StatusCode == 404 {
		return "", nil
	} else {
		return "", errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) GetTaskLogs(userId string, appName string, taskName string, statusFilter string) (types.TaskLogs, error) {
	url := client.Url + "/task/logs"
	getTaskLogsRequest := types.GetTaskLogsRequest{
		UserId:       userId,
		App:          appName,
		Task:         taskName,
		StatusFilter: statusFilter,
	}

	body, res, err := http.Call(url, "GET", getTaskLogsRequest, client.CredentialsProvider)
	if err != nil {
		return types.TaskLogs{
			Requests: []types.Request{},
		}, err
	}

	if res.StatusCode == 200 {
		var taskLogs types.TaskLogs
		if err != nil {
			return types.TaskLogs{
				Requests: []types.Request{},
			}, err
		}

		// TODO DON'T DOUBLE DESERIALIZE
		mapstructure.Decode(body, &taskLogs)
		return taskLogs, nil
	} else {
		// get res to string properly
		fmt.Println(res)
		err = errors.New("Error from server " + res.Status)
		return types.TaskLogs{
			Requests: []types.Request{},
		}, err
	}
}
