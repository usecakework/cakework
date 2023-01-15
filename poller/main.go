package main

import (
	"bytes"
	pb "cakework/poller/proto/cakework"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
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
var frontendUrl string

var accessToken, refreshToken string

func main() {
	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	flag.Parse()

	local = *localPtr

	var nc *nats.Conn

	if local == true {
		nc, _ = nats.Connect(nats.DefaultURL)
		fmt.Println("Local mode; connected to nats cluster: " + nats.DefaultURL)
		frontendUrl = "http://localhost:8080"
	} else {
		nc, _ = nats.Connect("cakework-nats-cluster.internal")
		fmt.Println("Non-local mode; connected to nats cluster: cakework-nats-cluster.internal")
		frontendUrl = "cakework-frontend.fly.dev"
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


		 // TODO delete this
		 fmt.Println("got a task run")
		 fmt.Println(taskRun)
         if err != nil {
			fmt.Println(err)
            // log.Fatal(err)
         }
         log.Printf("UserId: %s, App: %s, Task:%s, Parameters: %s, RequestId: %s, Status: %s, Result: %s\n", taskRun.UserId, taskRun.App, taskRun.Task, taskRun.Parameters, taskRun.RequestId, taskRun.Status, taskRun.Result)
         runTask(js, taskRun)
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
	var conn *grpc.ClientConn

	var endpoint string

	if local == true {
		endpoint = "localhost:50051"
	} else {
		endpoint = taskRun.UserId + "-" + taskRun.App + "-" + taskRun.Task + ".fly.dev:443" // TODO replace this with the actual name of the fly task (uuid)
	}
    // conn, err := grpc.Dial("shared-app-say-hello.fly.dev:443", grpc.WithInsecure())
	// endpoint := taskRun.UserId + "-" + taskRun.App + "-" + taskRun.Task + ".fly.dev:443" // TODO replace this with the actual name of the fly task (uuid)

    conn, err := grpc.Dial(endpoint, grpc.WithInsecure()) // TODO: don't create a new connection and client with every request; use a pool? 

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
		log.Println("Successfully submitted task to worker") // don't really need this

		// log.Printf("Successfully submitted task to worker:  %s", response.Result) // don't really need this
		// TODO: if fail, do not ack the request? but if we do so will the request get processed over and over again?
	}
	return nil
 }