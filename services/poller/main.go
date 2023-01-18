package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/auth"
	flyUtil "github.com/usecakework/cakework/lib/fly"
	flyApi "github.com/usecakework/cakework/lib/fly/api"
	pb "github.com/usecakework/cakework/poller/proto/cakework"
	"google.golang.org/grpc"
)

const (
	subSubjectName ="TASKS.created"
 
 )

 type TaskRun struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
	Task string `json:"task"`
    Parameters  string  `json:"parameters"`
	RequestId     string  `json:"requestId"`
	Status	string `json:"status"`
	Result  string  `json:"result"`
}

type UpdateStatusRequest struct {
	UserId     string  `json:"userId"`
	App	string `json:"app"`
    RequestId  string  `json:"requestId"`
	Status  string  `json:"status"`
}

var local bool
var verbose bool
var frontendUrl string
var flyMachineUrl string

var accessToken, refreshToken string
var fly *flyApi.Fly
var credentialsProvider auth.BearerStringCredentialsProvider

const NATS_URL = "cakework-nats-cluster.internal"

func main() {
	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	verbosePtr := flag.Bool("verbose", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	
	flag.Parse()

	local = *localPtr
	verbose = *verbosePtr

	var nc *nats.Conn

	var natsUrl string

	if local == true {
		natsUrl = nats.DefaultURL
		nc, _ = nats.Connect(nats.DefaultURL)
		fmt.Println("Local mode")
		frontendUrl = "http://localhost:8080"
		flyMachineUrl = "http://127.0.0.1:4280"
	} else {
		natsUrl = NATS_URL
		nc, _ = nats.Connect("cakework-nats-cluster.internal")
		fmt.Println("Non-local mode")
		frontendUrl = "cakework-frontend.fly.dev"
		flyMachineUrl = "http://_api.internal:4280"
	}

	fmt.Println("NATS url: " + natsUrl)
	fmt.Println("Frontend url: " + frontendUrl)
	fmt.Println("Fly Machine url: " + flyMachineUrl)

	if verbose {
		log.SetLevel(log.DebugLevel)
	} else{
		log.SetLevel(log.InfoLevel)
	}

	// Creates JetStreamContext
	js, err := nc.JetStream()
	checkErr(err)
   
	// Create Pull based consumer with maximum 128 inflight.
   // PullMaxWaiting defines the max inflight pull requests.
   go poll(js)
   gin.SetMode(gin.ReleaseMode)
   router := gin.Default()

   accessToken, refreshToken = getToken()
   credentialsProvider = auth.BearerStringCredentialsProvider{ Token: "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo" } // TODO remove this and rotate

   fly = flyApi.New("sahale", "http://127.0.0.1:4280", credentialsProvider) // TODO remove this secret for public launch
   router.Run(":8081")
}

func poll(js nats.JetStreamContext) {
	for {
		// Q: should we be creating a new pullsubscribe each time?
		sub, _ := js.PullSubscribe(subSubjectName, "submitted-tasks", nats.PullMaxWaiting(128))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	 
	    defer cancel()

    	msgs, _ := sub.Fetch(10, nats.Context(ctx))
      	for _, msg := range msgs {
			msg.Ack()
			var taskRun TaskRun
			err := json.Unmarshal(msg.Data, &taskRun)

			log.Info("Got task run")
			
			if err != nil {
				fmt.Println(err)
			}
			// log.Printf("UserId: %s, App: %s, Task:%s, Parameters: %s, RequestId: %s, Status: %s, Result: %s\n", taskRun.UserId, taskRun.App, taskRun.Task, taskRun.Parameters, taskRun.RequestId, taskRun.Status, taskRun.Result)
			if err := runTask(js, taskRun); err != nil { // TODO: handle error if RunTask throws an error
				log.Error("Error while processing task for " + taskRun.UserId + ", " + taskRun.App + ", " + taskRun.Task + ", " + taskRun.RequestId)
				log.Error(err)
			}
		}
   }
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// reviewOrder reviews the order and publishes ORDERS.approved event
func runTask(js nats.JetStreamContext, taskRun TaskRun) error {
	flyApp := flyUtil.GetFlyAppName(taskRun.UserId, taskRun.App, taskRun.Task)	

	image, err := db.GetLatestImage(flyApp)
	if err != nil {
		log.Error("Failed to get latest image to deploy")
		return err 
	}


	// spin up a new fly machine
	// get latest image so we know the version to spin up
	// every time we trigger a new deploy from the cli, we will update the the FlyMachine table
	// query for the latest created FlyMachine triggered by the cli and get the image from it
	// we don't update machines, we just spin up and spin down
	// use the image to spin up a new machine
	// once the spin up succeeds, parse the response to get the machine id 
	// submit request to the machine

	// so cli: 
	// spin up a new fly machine with source=CLI
	// insert into FlyMachine table via call to the frontend

	// TODO remove hardcoding
	image := "registry.fly.io/fly-machines:deployment-01GPYM48RWAP9GHWKWP0FNRE4D"
	cpus := 1
	memoryMB := 256

	// TODO hard code this 

	// TODO remoe!!!!!!!
	taskRun.UserId = "105349741723321386951"
	taskRun.App = "fly-machines"
	taskRun.Task = "say-hello"
	taskRun.RequestId = "my-request-id"

	/////
	// TODO get the latest created image from the FlyMachine table 
	flyApp := flyUtil.GetFlyAppName(taskRun.UserId, taskRun.App, taskRun.Task)	

	err := fly.NewMachine(flyApp, taskRun.RequestId, image, cpus, memoryMB)
	// TODO get response so we know what machine id to persist in frontend, as well as the machine id to invoke 

	if err != nil {
		log.Error(err)
		log.Error("Failed to deploy new Fly machine")
	}

	var conn *grpc.ClientConn

	var endpoint string

	if local == true {
		endpoint = "localhost:50051"
	} else {
		endpoint = "http://5683dd4b73d78e.vm.fly-machines.internal:50051"

		// endpoint = taskRun.UserId + "-" + taskRun.App + "-" + taskRun.Task + ".fly.dev:443" // TODO replace this with the actual name of the fly task (uuid)
	}
    // conn, err := grpc.Dial("shared-app-say-hello.fly.dev:443", grpc.WithInsecure())
	// endpoint := taskRun.UserId + "-" + taskRun.App + "-" + taskRun.Task + ".fly.dev:443" // TODO replace this with the actual name of the fly task (uuid)

    conn, err = grpc.Dial(endpoint, grpc.WithInsecure()) // TODO: don't create a new connection and client with every request; use a pool? 

	if err != nil {
		fmt.Printf("did not connect: %s", err)
		return err
		// TODO do something with the error; for example, fail the task
	}
	defer conn.Close()

	c := pb.NewCakeworkClient(conn)
	createReq := pb.Request{ Parameters: taskRun.Parameters, UserId: taskRun.UserId, App: taskRun.App, RequestId: taskRun.RequestId }
	_, errRunActivity := c.RunActivity(context.Background(), &createReq) // TODO: need to figure out how to expose the error that is thrown here (by the python code) to the users!!! 
	if errRunActivity != nil {
		// TODO check what type of error. possible to see if it's an rpc error?
		fmt.Println("Error Cakework RunActivity")

		fmt.Println(errRunActivity) // TODO log this as an error

		updateReq := UpdateStatusRequest {
			UserId: taskRun.UserId,
			App: taskRun.App,
			RequestId: taskRun.RequestId,
			Status: "FAILED",
		}

		jsonReq, err := json.Marshal(updateReq) // TODO handle possible error here 

		if err != nil {
			log.Fatal(err)
			fmt.Println(err)
		}
	
		// 2.
		client := &http.Client{}
		u, err := url.Parse(frontendUrl)
		if err != nil { fmt.Println(err) }
		u.Path = path.Join(u.Path, "update-status")

		// fmt.Println("calling url: " + u.String())
	
		// 3.

		req, err := http.NewRequest(http.MethodPatch, u.String(), bytes.NewBuffer(jsonReq))
		req.Header.Set("Content-Type", "application/json")
		
		// check that we have a non-expired access token
		if isTokenExpired(accessToken) {
			fmt.Println("Refreshing tokens")
			accessToken, refreshToken = refreshTokens(refreshToken)
			if accessToken == "" || refreshToken == "" {
				panic("Failed to refresh tokens")
			} else {
				fmt.Println("Refreshed tokens")
			}
		}

		req.Header.Set("Authorization", "Bearer " + accessToken)
		if err != nil {
			fmt.Println(err)
		}
	
		// 4.
		resp, err := client.Do(req)
		if err != nil {
			// log.Fatal(err)
			fmt.Println(err)
		} else {
			fmt.Println("Updated status to FAILED")
		}
	
		// 5.
		defer resp.Body.Close()
	
		// 6.
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// log.Fatal(err)
			fmt.Println(err)
		}
		log.Println(string(body))
	

		// TODO: need to log the error to a database so that the user can see if when they're querying for the status (and result?)
		return errRunActivity
		// instead of restarting the error by throwing a fatal, just do something with this. 
		// set the status to failed?
		// TODO need to be able to hook into frontend service to update the status

	} else {
		// successfully submitted; move to IN_PROGRESS
		// note: the fly python grpc worker probably still need to be able to update the status
		// what if this is updated to in progress but the python process sets to complete at the same time? should just let python deal with it.

		// note: can ignore the response from the worker for now
		// TODO: make sure don't print this out until we actually succeed
		log.Println("Successfully submitted task to worker") // don't really need this

		// log.Printf("Successfully submitted task to worker:  %s", response.Result) // don't really need this
		// TODO: if fail, do not ack the request? but if we do so will the request get processed over and over again?
	}
	return nil
 }