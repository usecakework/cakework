package frontendclient

import (
	"fmt"
	"os"

	"github.com/usecakework/cakework/lib/auth"
	fly "github.com/usecakework/cakework/lib/fly/cli"
	"github.com/usecakework/cakework/lib/http"
	"github.com/usecakework/cakework/lib/types"
)

type Client struct {
	Url string
	CredentialsProvider auth.CredentialsProvider
}

func New(url string, credentialsProvider auth.CredentialsProvider) *Client {
// func New(url string, accessToken string, refreshToken string, apiKey string) *Client {
	client := &Client{
		Url: url,
		CredentialsProvider: credentialsProvider,
	}

	return client
}

// TODO change all instances of appName to project
func (client *Client) CreateMachine(userId string, project string, task string, name string, machineId string, state string, image string) error {
	flyApp := fly.GetFlyAppName(userId, project, task)

	url := client.Url + "/create-machine"
	req := types.CreateMachineRequest{
		UserId: userId,
		Project: project,
		Task: task,
		FlyApp: flyApp,
		Name: name,
		MachineId: machineId,
		State: state,
		Image: image,
	}

	// TODO possibly improve error handling by checking status code here and returning an error message
	_, res := http.Call(url, "POST", req, client.CredentialsProvider)
	if res.StatusCode != 201 {
		fmt.Println("Failed to call Frontend to create machine")
		os.Exit(1)
	}
	// TODO: should have the inner processes propagate up the errors instead of exiting?
	return nil
}

// func (client *Client) GetUser(userId string, accessToken string, refreshToken string) *User {
// 	url := client.url + "/get-user"
// 	getUserRequest := GetUserRequest{
// 		UserId: userId,
// 	}
// 	jsonReq, err := json.Marshal(getUserRequest)
// 	checkOsExit(err)

// 	req, err := newRequestWithAuth("GET", url, bytes.NewBuffer(jsonReq))
// 	checkOsExit(err)

// 	_, body, res := callHttp(req)
// 	if res.StatusCode == 200 {
// 		userId := body["id"].(string)
// 		return &User{Id: userId}
// 	} else {
// 		fmt.Println("Error getting user details")
// 		fmt.Println(res)
// 		return nil
// 	}
	
// }

// func createUser(userId string) *User { // TODO change return type
// 	url := frontendURL + "/create-user"
// 	getUserRequest := CreateUserRequest{
// 		UserId: userId,
// 	}
// 	jsonReq, err := json.Marshal(getUserRequest)
// 	util.CheckOsExit(err)

// 	req, err := newRequestWithAuth("POST", url, bytes.NewBuffer(jsonReq))
// 	util.CheckOsExit(err)

// 	_, body, res := callHttp(req)
// 	if res.StatusCode == 201 {
// 		userId := body["id"].(string)
// 		return &User{Id: userId}
// 	} else {
// 		fmt.Println("Error creating user")
// 		fmt.Println(res)
// 		return nil
// 	}
// }

// func createClientToken(userId string, name string) *ClientToken { // TODO change return type
// 	url := frontendURL + "/create-client-token"
// 	createTokenReq := CreateTokenRequest{
// 		UserId: userId,
// 		Name:   name,
// 	}
// 	jsonReq, err := json.Marshal(createTokenReq)
// 	util.CheckOsExit(err)

// 	req, err := newRequestWithAuth("POST", url, bytes.NewBuffer(jsonReq))
// 	checkOsExit(err)

// 	_, body, res := callHttp(req)
// 	if res.StatusCode == 201 {
// 		token := body["token"].(string)
// 		return &ClientToken{Token: token}
// 	} else {
// 		fmt.Println("Error creating client token")
// 		fmt.Println(res)
// 		return nil
// 	}
// }

// // TODO return errors
// func getRequestStatus(userId string, requestId string) string {
// 	url := frontendURL + "/get-status"
// 	getStatusRequest := GetStatusRequest{
// 		UserId:    userId,
// 		RequestId: requestId,
// 	}
// 	jsonReq, err := json.Marshal(getStatusRequest)
// 	checkOsExit(err)

// 	req, err := newRequestWithAuth("GET", url, bytes.NewBuffer(jsonReq))
// 	checkOsExit(err)

// 	_, body, res := callHttp(req)
// 	if res.StatusCode == 200 {
// 		status := body["status"].(string)
// 		return status
// 	} else if res.StatusCode == 404 {
// 		fmt.Println("Request ID " + requestId + " does not exist")
// 		return ""
// 	} else {
// 		checkOsExit(errors.New("Error getting request status, got an" + res.Status))
// 		return ""
// 	}
// }

// func getTaskLogs(userId string, appName string, taskName string, statuses []string) TaskLogs {
// 	url := frontendURL + "/task/logs"
// 	getTaskLogsRequest := GetTaskLogsRequest{
// 		UserId: userId,
// 		App:    appName,
// 		Task:   taskName,
// 	}
// 	jsonReq, err := json.Marshal(getTaskLogsRequest)
// 	checkOsExit(err)

// 	req, err := newRequestWithAuth("GET", url, bytes.NewBuffer(jsonReq))
// 	checkOsExit(err)

// 	res, err := http.DefaultClient.Do(req)
// 	checkOsExit(err)

// 	if res.StatusCode == 200 {
// 		var taskLogs TaskLogs
// 		bodybutbetter, err := io.ReadAll(res.Body)
// 		if err != nil {
// 			checkOsExit(errors.New("Error running task " + appName + "/" + taskName))
// 		}

// 		json.Unmarshal(bodybutbetter, &taskLogs)
// 		return taskLogs
// 	} else {
// 		// get res to string properly
// 		fmt.Println(res)
// 		checkOsExit(errors.New("Error running task " + appName + "/" + taskName))
// 		return TaskLogs{
// 			Requests: []Request{},
// 		}
// 	}
// }

// call a frontend API, using the input request
// func (client *Client) Call(frontendReq interface{}, route string, method string) (map[string]interface{}, *http.Response) {
// 	jsonReq, err := json.Marshal(frontendReq)
// 	util.CheckOsExit(err)
	
// 	req, err := cli.NewRequestWithAuth("POST", url, bytes.NewBuffer(jsonReq))
// 	util.CheckOsExit(err)

// 	util.callHttp
// }
