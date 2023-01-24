package frontendclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

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
func (client *Client) CreateMachine(userId string, project string, task string, name string, machineId string, state string, image string, source string) error {
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
		Source:    source,
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
	defer res.Body.Close()

	if res.StatusCode == 200 {
		status := body["status"].(string)
		return status, nil
	} else if res.StatusCode == 404 {
		return "", nil
	} else {
		return "", errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) GetRequestLogs(userId string, requestId string) (*types.RequestLogs, error) {
	url := client.Url + "/request/logs"
	getRequestLogsRequest := types.GetRequestLogsRequest{
		UserId:    userId,
		RequestId: requestId,
	}

	res, err := http.CallV2(url, "GET", getRequestLogsRequest, client.CredentialsProvider)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		var requestLogs types.RequestLogs

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

func (client *Client) GetTaskLogs(userId string, appName string, taskName string, statusFilter string) (types.TaskLogs, error) {
	url := client.Url + "/task/logs"
	getTaskLogsRequest := types.GetTaskLogsRequest{
		UserId:       userId,
		App:          appName,
		Task:         taskName,
		StatusFilter: statusFilter,
	}

	res, err := http.CallV2(url, "GET", getTaskLogsRequest, client.CredentialsProvider)
	if err != nil {
		return types.TaskLogs{
			Requests: []types.Request{},
		}, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		var taskLogs types.TaskLogs
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return types.TaskLogs{
				Requests: []types.Request{},
			}, err
		}

		json.Unmarshal(body, &taskLogs)
		return taskLogs, nil
	} else {
		// get res to string properly
		fmt.Println(res)
		err = errors.New("Server Error " + res.Status)
		return types.TaskLogs{
			Requests: []types.Request{},
		}, err
	}
}

func (client *Client) UpdateStatus(userId string, app string, requestId string, status string) error {
	url := client.Url + "/update-status"
	req := types.UpdateStatusRequest{
		UserId:    userId,
		App: app,
		RequestId: requestId,
		Status: status,
	}

	_, res, err := http.Call(url, "PATCH", req, client.CredentialsProvider)
	if err != nil {
		fmt.Println(res) // TODO should be logging instead
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 || res.StatusCode == 201 {
		return nil
	} else {
		return errors.New("Error getting request status from server. " + res.Status)
	}
}

func (client *Client) UpdateMachineId(userId string, app string, requestId string, machineId string) error {
	url := client.Url + "/update-machine-id"
	req := types.UpdateMachineId{
		UserId:    userId,
		App: app,
		RequestId: requestId,
		MachineId: machineId,
	}

	fmt.Println("about to call frontend to update machine id") // TODO delete
	_, res, err := http.Call(url, "PATCH", req, client.CredentialsProvider)
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
	url := client.Url + "/get-cli-secrets"
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