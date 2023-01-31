package frontendclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/usecakework/cakework/lib/auth"
	fly "github.com/usecakework/cakework/lib/fly"
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
func (client *Client) CreateMachine(userId string, project string, task string, name string, machineId string, state string, image string, source string) error {
	flyApp := fly.GetFlyAppName(userId, project, task)

	url := client.Url + "/projects/" + project + "/tasks/" + task + "/machines"
	req := types.CreateMachineRequest{
		UserId:    userId,
		Project:   project,
		Task:      task,
		FlyApp:    flyApp,
		Name:      name,
		MachineId: machineId,
		State:     state,
		Image:     image,
		Source:    source,
	}

	res, err := http.CallV2(url, "POST", req, client.CredentialsProvider)
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
	url := client.Url + "/users/" + userId
	getUserRequest := types.GetUserRequest{
		UserId: userId,
	}

	res, err := http.CallV2(url, "GET", getUserRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		var user types.User
		json.Unmarshal(body, &user)
		return &user, nil
	} else if res.StatusCode == 404 {
		return nil, nil
	} else {
		return nil, errors.New("Error getting user details." + res.Status)
	}
}

func (client *Client) CreateUser(userId string) (*types.User, error) { // TODO change return type
	url := client.Url + "/users"
	createUserRequest := types.CreateUserRequest{
		UserId: userId,
	}

	res, err := http.CallV2(url, "POST", createUserRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 200 || res.StatusCode == 201 {
		var user types.User
		if err := json.Unmarshal(body, &user); err != nil {
			return nil, err
		}
		return &user, nil
	} else {
		return nil, errors.New("Error creating a new user." + res.Status)
	}
}

func (client *Client) CreateClientToken(userId string, name string) (*types.ClientToken, error) { // TODO change return type
	url := client.Url + "/client-tokens"
	createTokenReq := types.CreateTokenRequest{
		UserId: userId,
		Name:   name,
	}

	res, err := http.CallV2(url, "POST", createTokenReq, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 201 {
		var tok *types.ClientToken
		json.Unmarshal(body, &tok)
		return tok, nil
	} else {
		return nil, errors.New("Error creating client token" + res.Status)
	}
}

func (client *Client) GetRunRequestStatus(userId string, runId string) (string, error) {
	url := client.Url + "/runs/" + runId + "/status"
	getStatusRequest := types.GetRunStatusRequest{
		UserId:    userId,
		RunId: runId,
	}

	res, err := http.CallV2(url, "GET", getStatusRequest, client.CredentialsProvider)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", nil
	}

	if res.StatusCode == 200 {
		var status types.GetRunStatusResponse
		if err := json.Unmarshal(body, &status); err != nil {
			return "", err
		}
		return status.Status, nil
	} else if res.StatusCode == 404 {
		return "", nil
	} else {
		return "", errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) GetRunLogs(userId string, runId string) (*types.RunLogs, error) {
	url := client.Url + "/runs/" + runId + "/logs"
	getRunLogsRequest := types.GetRunLogsRequest{
		UserId:    userId,
		RunId: runId,
	}

	res, err := http.CallV2(url, "GET", getRunLogsRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		var requestLogs types.RunLogs

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(body, &requestLogs)

		return &requestLogs, nil
	} else if res.StatusCode == 404 {
		return nil, nil
	} else {
		// get res to string properly
		fmt.Println(res)
		return nil, errors.New("Server error: " + res.Status)
	}
}

func (client *Client) GetTaskLogs(userId string, projectName string, taskName string, statusFilter string) (types.TaskLogs, error) {
	url := client.Url + "/projects/" + projectName + "/tasks/" + taskName + "/logs"
	getTaskLogsRequest := types.GetTaskLogsRequest{
		UserId:       userId,
		Project:      projectName,
		Task:         taskName,
		StatusFilter: statusFilter,
	}

	res, err := http.CallV2(url, "GET", getTaskLogsRequest, client.CredentialsProvider)
	if err != nil {
		return types.TaskLogs{
			Runs: []types.Run{},
		}, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		var taskLogs types.TaskLogs
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return types.TaskLogs{
				Runs: []types.Run{},
			}, err
		}

		json.Unmarshal(body, &taskLogs)
		return taskLogs, nil
	} else {
		// get res to string properly
		fmt.Println(res)
		err = errors.New("Server Error " + res.Status)
		return types.TaskLogs{
			Runs: []types.Run{},
		}, err
	}
}

func (client *Client) UpdateRunStatus(userId string, project string, runId string, status string) error {
	url := client.Url + "/runs/" + runId + "/status"
	req := types.UpdateRunStatusRequest{
		RunId: runId,
		Status:    status,
	}

	res, err := http.CallV2(url, "POST", req, client.CredentialsProvider)
	if err != nil {
		fmt.Println(res) // TODO should be logging instead
		return err
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 || res.StatusCode == 201 {
		return nil
	} else {
		return errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) UpdateMachineId(userId string, project string, runId string, machineId string) error {
	url := client.Url + "/runs/" + runId + "/machineId"
	req := types.UpdateMachineIdRequest{
		UserId:    userId,
		Project:   project,
		RunId:     runId,
		MachineId: machineId,
	}

	res, err := http.CallV2(url, "POST", req, client.CredentialsProvider)
	if err != nil {
		fmt.Println("Got error calling frontend to update machine id")
		fmt.Println(res) // TODO should be logging instead. most likely this is nil
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 || res.StatusCode == 201 {
		return nil
	} else {
		return errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) GetCLISecrets() (*types.CLISecrets, error) {
	url := client.Url + "/cli-secrets"
	var secrets types.CLISecrets

	res, err := http.CallV2(url, "GET", nil, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		// var data map[string]interface{}
		// err = json.NewDecoder(body).Decode(&data)

		json.Unmarshal(body, &secrets)
		return &secrets, nil
	} else if res.StatusCode == 404 {
		return nil, errors.New("404 not found from frontend")
	} else {
		fmt.Println(res)
		return nil, errors.New("Server error: " + res.Status)
	}
}
