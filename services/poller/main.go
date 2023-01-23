package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	"github.com/usecakework/cakework/lib/types"
	pb "github.com/usecakework/cakework/poller/proto/cakework"
	"google.golang.org/grpc"
)

const (
	subSubjectName    = "TASKS.created"
	DEFAULT_CPU       = 1
	DEFAULT_MEMORY_MB = 256
)

type UpdateStatusRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	Status    string `json:"status"`
	MachineId string `json:"machineId"`
}

var local bool
var verbose bool
var frontendUrl string
var flyMachineUrl string
var DSN string

var accessToken, refreshToken string
var fly *flyApi.Fly
var credentialsProvider auth.BearerStringCredentialsProvider
var db *sql.DB

//go:embed .env
var envFile []byte

// this isn't really needed, but vscode auto removes the import for embed if it's not referenced
//
//go:embed fly.toml
var flyConfig embed.FS

func main() {
	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally")     // can pass go run main.go -local
	verbosePtr := flag.Bool("verbose", false, "boolean which if true runs the poller locally") // can pass go run main.go -local

	flag.Parse()

	local = *localPtr
	verbose = *verbosePtr

	stage := os.Getenv("STAGE")
	if stage == "" {
		log.Fatal("Failed to get stage from environment variable")
	} else {
		log.Info("Got stage: " + stage)
	}

	if stage == "dev" {
		viper.SetConfigType("dotenv")
		err := viper.ReadConfig(bytes.NewBuffer(envFile))

		if err != nil {
			fmt.Println(fmt.Errorf("%w", err))
			os.Exit(1)
		}
	} else {
		viper.SetConfigType("env")
	}

	var nc *nats.Conn

	NATS_CLUSTER := viper.GetString("NATS_CLUSTER")
	nc, _ = nats.Connect(NATS_CLUSTER)
	frontendUrl = viper.GetString("FRONTEND_URL")
	flyMachineUrl = viper.GetString("FLY_MACHINES_URL")
	DSN = viper.GetString("DB_CONN_STRING")

	fmt.Println("NATS url: " + natsUrl)
	fmt.Println("Frontend url: " + frontendUrl)
	fmt.Println("Fly Machine url: " + flyMachineUrl)

	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
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
	FLY_ACCESS_TOKEN := viper.GetString("FLY_ACCESS_TOKEN")
	credentialsProvider = auth.BearerStringCredentialsProvider{Token: FLY_ACCESS_TOKEN}

	fly = flyApi.New("sahale", flyMachineUrl, credentialsProvider) // TODO remove this secret for public launch

	db, err = sql.Open("mysql", DSN)
	if err != nil {
		log.Error("Failed to open database connection")
		log.Error(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)

	defer db.Close()

	if err != nil {
		log.Error("Failed to initialize database connection")
		log.Error(err)
	}
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
			var req types.Request
			err := json.Unmarshal(msg.Data, &req)

			log.Infof("Got request: " + req.UserId + ", " + req.App + ", " + req.Task + ", " + req.RequestId)

			if err != nil {
				fmt.Println(err)
			}
			if err := runTask(js, req); err != nil { // TODO: handle error if RunTask throws an error
				log.Error("Error while processing task for " + req.UserId + ", " + req.App + ", " + req.Task + ", " + req.RequestId)
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
func runTask(js nats.JetStreamContext, req types.Request) error {
	flyApp := flyUtil.GetFlyAppName(req.UserId, req.App, req.Task)

	image, err := GetLatestImage(flyApp, db)
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

	// in cli:
	// spin up a new fly machine with source=CLI
	// insert into FlyMachine table via call to the frontend

	// TODO remove hardcoding
	var cpu int
	var memoryMB int
	if req.CPU == 0 {
		cpu = DEFAULT_CPU
	} else {
		cpu = req.CPU
	}
	if req.MemoryMB == 0 {
		memoryMB = DEFAULT_MEMORY_MB
	} else {
		memoryMB = req.MemoryMB
	}

	// TODO remove these dummy values
	// req.UserId = "105349741723321386951"
	// taskRun.App = "fly-machines"
	// taskRun.Task = "say-hello"
	// taskRun.RequestId = "my-request-id"

	log.Infof("Spinning up a machine with parameters: %s, %s, %s, %d, %d", flyApp, req.RequestId, image, cpu, memoryMB)
	machineConfig, err := fly.NewMachine(flyApp, req.RequestId, image, cpu, memoryMB)
	if err != nil {
		log.Error("Failed to spin up new Fly machine")
		return err
	}

	if machineConfig.MachineId == "" {
		return errors.New("Machine id of spun up machine is null; error occurred somewhere")
	}

	// wait for machine to get to started status
	desiredState := "started"
	err = fly.Wait(flyApp, machineConfig.MachineId, desiredState)
	if err != nil {
		log.Error(err)
		return errors.New("Machine failed to reach " + desiredState + " in 60 seconds. Needs longer timeout, or an error occurred")
		// TODO also fetch and print out current state, for debugging
	}

	// TODO get response so we know what machine id to persist in frontend, as well as the machine id to invoke
	// TODO insert into the FlyMachine table for tracking. For now, don't need to bother

	var conn *grpc.ClientConn

	var endpoint string

	if local == true {
		// endpoint = "localhost:50051" // not yet supported; for running the grpc server locally
		endpoint = machineConfig.MachineId + ".vm." + flyApp + ".internal:50051"
	} else {
		endpoint = machineConfig.MachineId + ".vm." + flyApp + ".internal:50051"
	}

	log.Info("Attempting to send request to machine endpoint: " + endpoint)
	conn, err = grpc.Dial(endpoint, grpc.WithInsecure()) // TODO: don't create a new connection and client with every request; use a pool?

	if err != nil {
		fmt.Printf("did not connect: %s", err)
		return err
		// TODO do something with the error; for example, fail the task
	}
	defer conn.Close()

	c := pb.NewCakeworkClient(conn)
	createReq := pb.Request{Parameters: req.Parameters, UserId: req.UserId, App: req.App, RequestId: req.RequestId}
	_, errRunActivity := c.RunActivity(context.Background(), &createReq) // TODO: need to figure out how to expose the error that is thrown here (by the python code) to the users!!!
	if errRunActivity != nil {
		// TODO check what type of error. possible to see if it's an rpc error?
		fmt.Println("Error Cakework RunActivity")

		fmt.Println(errRunActivity) // TODO log this as an error

		updateReq := UpdateStatusRequest{
			UserId:    req.UserId,
			App:       req.App,
			RequestId: req.RequestId,
			Status:    "FAILED",
		}

		jsonReq, err := json.Marshal(updateReq) // TODO handle possible error here

		if err != nil {
			log.Fatal(err)
			fmt.Println(err)
		}

		// 2.
		client := &http.Client{}
		u, err := url.Parse(frontendUrl)
		if err != nil {
			fmt.Println(err)
		}
		u.Path = path.Join(u.Path, "update-status")

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

		req.Header.Set("Authorization", "Bearer "+accessToken)
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
