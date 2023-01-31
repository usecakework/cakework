package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	fly "github.com/usecakework/cakework/lib/fly"
	cwHttp "github.com/usecakework/cakework/lib/http"
	"github.com/usecakework/cakework/lib/types"
)

const (
	streamName     = "RUNS"
	streamSubjects = "RUNS.*"
	subjectName    = "RUNS.created"
)

var db *sql.DB
var err error

var js nats.JetStreamContext
var local bool

// CustomClaimsExample contains custom data we want from the token.
type CustomClaimsExample struct {
	Scope string `json:"scope"`
}

func (c *CustomClaimsExample) Validate(ctx context.Context) error {
	return nil
}

var customClaims = func() validator.CustomClaims {
	return &CustomClaimsExample{}
}

var stage string

// this isn't really needed, but vscode auto removes the import for embed if it's not referenced
//
//go:embed fly.toml
var flyConfig embed.FS

func main() {
	verbosePtr := flag.Bool("verbose", false, "boolean which if true runs the poller locally")

	flag.Parse()

	verbose := *verbosePtr

	if verbose {
		log.Info("verbose=true")
		log.SetLevel(log.DebugLevel)
	} else {
		log.Info("verbose=false")
		log.SetLevel(log.InfoLevel)
	}

	stage := os.Getenv("STAGE")
	if stage == "" {
		log.Fatal("Failed to get stage from environment variable")
	} else {
		log.Info("Got stage: " + stage)
	}

	if stage == "dev" {
		viper.SetConfigType("dotenv")
		viper.SetConfigFile(".env")
		err := viper.ReadInConfig()

		if err != nil {
			fmt.Println(fmt.Errorf("%w", err))
			os.Exit(1)
		}
	} else {
		viper.SetConfigType("env")
		viper.AutomaticEnv()
	}

	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	flag.Parse()

	local = *localPtr

	var nc *nats.Conn

	NATS_CLUSTER := viper.GetString("NATS_CLUSTER")
	nc, _ = nats.Connect(NATS_CLUSTER)
	fmt.Println("Non-local mode; connected to nats cluster: " + NATS_CLUSTER)

	// Creates JetStreamContext
	js, err = nc.JetStream()
	checkErr(err)

	// Creates stream - note: should we need to do this every time? will this cause old stuff to be lost
	// Q: durability of the stream? where to store it? if reboot will things get lost?
	err = createStream(js)
	checkErr(err)

	DB_CONN_STRING := viper.GetString("DB_CONN_STRING")
	// Open the connection
	db, err = sql.Open("mysql", DB_CONN_STRING)
	if err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())
	router.Use(ginBodyLogMiddleware)
	router.Use(guidMiddleware())
	router.Use(cors.Default())

	// The issuer of our token.
	AUTH0_URL := viper.GetString("AUTH0_URL")
	issuerURL, _ := url.Parse(AUTH0_URL)

	// The audience of our token.
	audience := "https://cakework-frontend.fly.dev" // TODO put into .env

	provider := jwks.NewCachingProvider(issuerURL, time.Duration(5*time.Minute))

	jwtValidator, _ := validator.New(provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{audience},
		validator.WithCustomClaims(customClaims),
	)

	jwtMiddleware := jwtmiddleware.New(jwtValidator.ValidateToken)

	jwtProtectedGroup := router.Group("", adapter.Wrap(jwtMiddleware.CheckJWT))
	{
		// external scope means the CLI, task sdk and client sdk can call
		// user-initiated
		// TODO cache the token to user mappings
		jwtProtectedGroup.POST("/projects/:project/tasks/:task/runs", handleRun, jwtTokenMiddleware("external")) // header: bearer token. TODO: get userId from bearer token
		jwtProtectedGroup.GET("/runs/:runId/status", handleGetRunStatus, jwtTokenMiddleware("external"))         // header: bearer token
		jwtProtectedGroup.GET("/runs/:runId/result", handleGetRunResult, jwtTokenMiddleware("external"))         // header: bearer token
		jwtProtectedGroup.POST("/client-tokens", handleCreateClientToken, jwtTokenMiddleware("external"))        // header: bearer token

		// uses user creds to call, but initiated on behalf of the user by the CLI
		jwtProtectedGroup.POST("/users", handleCreateUser, jwtTokenMiddleware("external")) // only called by internal. TODO refactor so that we don't trigger this from the cli using the user's creds
		jwtProtectedGroup.GET("/user-from-client-token", handleGetUserFromClientToken, jwtTokenMiddleware("external"))
		jwtProtectedGroup.GET("/users/:userId", handleGetUser, jwtTokenMiddleware("external"))
		jwtProtectedGroup.GET("/cli-secrets", handleGetCLISecrets, jwtTokenMiddleware("external")) // TODO remove this in the future once we have our own build server
		jwtProtectedGroup.GET("/projects/:project/tasks/:task/logs", handleGetTaskLogs, jwtTokenMiddleware("external"))
		jwtProtectedGroup.GET("/runs/:runId/logs", handleGetRunLogs, jwtTokenMiddleware("external"))
		jwtProtectedGroup.POST("/projects/:project/tasks/:task/machines", handleCreateMachine, jwtTokenMiddleware("external"))

		// only internal services can call
		jwtProtectedGroup.POST("/runs/:runId/status", handleUpdateRunStatus, jwtTokenMiddleware("internal"))
		jwtProtectedGroup.POST("/runs/:runId/result", handleUpdateRunResult, jwtTokenMiddleware("internal"))
		jwtProtectedGroup.POST("/runs/:runId/machineId", handleUpdateMachineId, jwtTokenMiddleware("internal"))
	}

	apiKeyProtectedGroup := router.Group("/client", apiKeyMiddleware())
	{
		apiKeyProtectedGroup.GET("/user-from-client-token", handleGetUserFromClientToken) // user never actually invokes this, but client needs to
		apiKeyProtectedGroup.GET("/runs/:runId/status", handleGetRunStatus)
		apiKeyProtectedGroup.GET("/runs/:runId/result", handleGetRunResult)
		apiKeyProtectedGroup.POST("/projects/:project/tasks/:task/runs", handleRun)
	}

	router.Run()
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func ginBodyLogMiddleware(c *gin.Context) {
	blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = blw
	c.Next()
	statusCode := c.Writer.Status()
	if statusCode >= 400 {
		//ok this is an request with error, let's make a record for it
		// now print body (or log in your preferred way)
		log.Error("Response body: " + blw.body.String())
	}
}

func guidMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.New()
		c.Set("uuid", uuid)
		log.Printf("Request started: %s\n", uuid)
		log.Debug(cwHttp.PrettyPrintRequest(c.Request))
		log.Debug(c.Request.Host)

		c.Next()
		log.Printf("Request finished: %s\n", uuid)
	}
}

func handleGetRunLogs(c *gin.Context) {
	var newGetRunLogsRequest types.GetRunLogsRequest

	if err := c.BindJSON(&newGetRunLogsRequest); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusBadRequest, "Issue with parsing request to json")
		return
	}

	runDetails, err := getRun(db, newGetRunLogsRequest.RunId)

	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, "Sorry :( something broke, come talk to us")
		return
	}

	if runDetails == nil {
		c.IndentedJSON(http.StatusNotFound, "Run "+newGetRunLogsRequest.RunId+" does not exist.")
		return
	}

	runId := runDetails.RunId
	userId := runDetails.UserId
	project := runDetails.Project
	taskName := runDetails.Task
	machineId := runDetails.MachineId

	logs, err := getRunLogs(userId, project, taskName, machineId, runId)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, "Sorry :( something broke, come talk to us")
		return
	}

	c.IndentedJSON(http.StatusOK, logs)
	return
}

func handleGetTaskLogs(c *gin.Context) {
	var newGetTaskLogsRequest types.GetTaskLogsRequest

	if err := c.BindJSON(&newGetTaskLogsRequest); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusBadRequest, "Issue with parsing request to json")
		return
	}

	userId := newGetTaskLogsRequest.UserId
	app := newGetTaskLogsRequest.Project
	task := newGetTaskLogsRequest.Task
	statusFilter := newGetTaskLogsRequest.StatusFilter

	taskLogs, err := GetTaskLogs(db, userId, app, task, statusFilter)

	// return task not found properly
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, "Sorry :( something broke, come talk to us")
		return
	}

	c.IndentedJSON(http.StatusOK, taskLogs)
}

func getStatus(c *gin.Context) {
	var newGetStatusRequest types.GetRunStatusRequest

	if err := c.BindJSON(&newGetStatusRequest); err != nil {
		log.Error(err)
		return
	}

	request, err := getRun(db, newGetStatusRequest.RunId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + newGetStatusRequest.RunId + " not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		}
	} else {
		response := types.GetRunStatusResponse{
			Status: request.Status,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func handleGetCLISecrets(c *gin.Context) {
	FLY_ACCESS_TOKEN := viper.GetString("FLY_ACCESS_TOKEN")
	if FLY_ACCESS_TOKEN == "" {
		log.Info("FLY_ACCESS_TOKEN shouldn't be null")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "FLY_ACCESS_TOKEN shouldn't be null"})
		return
	}

	secrets := types.CLISecrets{
		FLY_ACCESS_TOKEN: FLY_ACCESS_TOKEN,
	}

	c.IndentedJSON(http.StatusOK, secrets)
}

func getResult(c *gin.Context) {
	var newGetResultRequest types.GetRunResultRequest

	if err := c.BindJSON(&newGetResultRequest); err != nil {
		log.Error(err)
		return
	}

	request, err := getRun(db, newGetResultRequest.RunId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + newGetResultRequest.RunId + " not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		}
	} else {
		response := types.GetRunResultResponse{
			Result: request.Result,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func handleUpdateRunStatus(c *gin.Context) {
	var request types.UpdateRunStatusRequest

	if err := c.BindJSON(&request); err != nil {
		log.Error(err)
		return
	}

	stmt, err := db.Prepare("UPDATE Run SET status = ? WHERE RunId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.Status, request.RunId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		log.Error("nothing was updated")
		c.Status(http.StatusNotFound)
		return
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func handleUpdateRunResult(c *gin.Context) {
	var request types.UpdateRunResultRequest

	if err := c.BindJSON(&request); err != nil {
		log.Error(err)
		return
	}

	stmt, err := db.Prepare("UPDATE Run SET result = ? WHERE RunId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.Result, request.RunId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		log.Error("nothing was updated")
		c.Status(http.StatusNotFound)
		return
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func handleUpdateMachineId(c *gin.Context) {
	var request types.UpdateMachineIdRequest

	if err := c.BindJSON(&request); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
		return
	}

	// TODO verify that we aren't overwriting anything
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	stmt, err := db.Prepare("UPDATE Run SET machineId = ? WHERE RunId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.MachineId, request.RunId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		log.Error("nothing was updated")
		c.Status(http.StatusNotFound)
		return
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		} else {
			c.Status(http.StatusOK)
		}
	}
}

// no need to add scope checking here, as this is not directly invoked by a route
func enqueue(req types.Run) error {
	reqJSON, _ := json.Marshal(req)

	_, err := js.Publish(subjectName, reqJSON)
	if err != nil {
		log.Error("error while publishing")
		log.Error(err)
		return err // TODO return a human readable error
	} else {
		log.Printf("Request with RunId:%s has been published\n", req.RunId)
		query := "INSERT INTO `Run` (`RunId`, `userId`, `project`, `task`, `parameters`, `status`, `cpu`, `memory`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		insertResult, err := db.ExecContext(context.Background(), query, req.RunId, req.UserId, req.Project, req.Task, req.Parameters, req.Status, req.CPU, req.Memory)
		if err != nil {
			log.Error("Failed to insert : %s", err)
			return err
		}
		_, err = insertResult.LastInsertId()
		if err != nil {
			return err
		}
	}
	return nil
}

// createStream creates a stream by using JetStreamContext
func createStream(js nats.JetStreamContext) error {
	// Check if the ORDERS stream already exists; if not, create it.
	stream, err := js.StreamInfo(streamName)
	if err != nil {
		log.Println(err)
	}
	if stream == nil {
		log.Printf("creating stream %q and subjects %q", streamName, streamSubjects)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamSubjects},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// TODO: for the client token, add the scopes for submitting a new task, getting status, getting result if we move this to auth0?
// if the frontend api is locked down now, how will the client call the frontend?
func handleCreateClientToken(c *gin.Context) {
	var newRequest types.CreateClientTokenRequest
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Error("Failed to generate random token")
		log.Error(err) // TODO return a custom http response
	}

	token := hex.EncodeToString(b)

	if err := c.BindJSON(&newRequest); err != nil {
		log.Error(err)
		return
	}

	tokenId := (uuid.New()).String()
	updatedAt := time.Now()

	clientToken := types.ClientToken{
		Token: token,
	}

	query := "INSERT INTO `ClientToken` (`id`, `name`, `token`, `userId`, `updatedAt`) VALUES (?, ?, ?, ?, ?)"
	insertResult, err := db.ExecContext(context.Background(), query, tokenId, newRequest.Name, token, newRequest.UserId, updatedAt)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
		return
	} else {
		c.IndentedJSON(http.StatusCreated, clientToken)
	}
}

func handleCreateUser(c *gin.Context) {
	var newRequest types.CreateUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	}

	userId := newRequest.UserId

	newUser := types.User{
		Id: userId,
	}

	query := "INSERT INTO `User` (`id`) VALUES (?)"
	insertResult, err := db.ExecContext(context.Background(), query, newUser.Id)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		// log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?
		log.Printf("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, newUser)
	}
}

func handleGetUserFromClientToken(c *gin.Context) {

	// fetch the client token by the token value
	// return the user
	var newRequest types.GetUserByClientTokenRequest

	if err := c.BindJSON(&newRequest); err != nil {
		log.Error(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	var user types.User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", newRequest.Token).Scan(&user.Id)
	if err != nil {
		log.Error(err)
		if err.Error() == sql.ErrNoRows.Error() {
			c.IndentedJSON(http.StatusBadRequest, "Please provide a valid client token.")
			return
		} else {
			log.Debug("TODO catch the specific error (such as access denied)")
			c.IndentedJSON(http.StatusInternalServerError, "Something went wrong :( Please contact us.")
			return
		}
	} else {
		c.IndentedJSON(http.StatusOK, user)
	}
}

func handleGetUser(c *gin.Context) {
	var newRequest types.GetUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		log.Error(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	userId := newRequest.UserId

	// TODO use the userId and app
	var user types.User
	err = db.QueryRow("SELECT id FROM User where id = ?", userId).Scan(&user.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "user with id not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
			return
		}
		// log.Fatalf("impossible to fetch : %s", err) // we shouldn't exit??? or will this only kill the current thing? TODO test this behavior
	} else {
		c.IndentedJSON(http.StatusOK, user)
	}
}

// TODO put into a separate package. Can have the main.go invoke this as well
func getUserFromAPIKey(apiKey string) (*types.User, error) {
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?
	var user types.User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", apiKey).Scan(&user.Id)
	if err != nil && user.Id != "" {
		return nil, err
	} else {
		return &user, nil
	}
}

func handleCreateMachine(c *gin.Context) {
	var req types.CreateMachineRequest

	if err := c.BindJSON(&req); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// TODO put this into a middleware
	userId := req.UserId
	project := req.Project
	task := req.Task
	flyApp := fly.GetFlyAppName(userId, project, task)

	query := "INSERT INTO `FlyMachine` (`userId`, `project`, `task`, `flyApp`, `name`, `machineId`, `state`, `image`, `source`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertResult, err := db.ExecContext(context.Background(), query, userId, project, task, flyApp, req.Name, req.MachineId, req.State, req.Image, req.Source)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
		return
	} else {
		log.Info("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, req)
	}
}

// new refactor using REST API patterns
// TODO take i the project id
func handleGetRunStatus(c *gin.Context) {
	runId := c.Param("runId")
	// get project and user id from the headers
	// TODO cache the info about the user id

	// apiKey := c.Request.Header.Get("X-Api-Key") // TODO refactor to middleware
	// userId, err := getUserFromAPIKey(apiKey)
	// if err != nil {
	// 	c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"})
	// 	return
	// }

	// TODO add userId and project id to the requestDetails call

	// project := c.Request.Header.Get("project")

	request, err := getRun(db, runId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + runId + " not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		}
	} else {
		log.Debug("Got a request: ")
		log.Debug(request)
		c.IndentedJSON(http.StatusOK, request.Status)
		return
	}
}

// new refactor using REST API patterns
// TODO take i the project id
func handleGetRunResult(c *gin.Context) {
	runId := c.Param("runId")

	request, err := getRun(db, runId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + runId + " not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		}
	} else {
		log.Debug("Got a request: ")
		log.Debug(request)
		c.IndentedJSON(http.StatusOK, request.Result)
		return
	}
}

// TODO have this throw error?
func handleRun(c *gin.Context) {
	project := c.Param("project")
	task := c.Param("task")

	// TODO check if app exists; if not, throw an error
	var runReq types.RunRequest
	if err := c.BindJSON(&runReq); err != nil {
		log.Error(err)
		return
	}

	// get user id and project from the headers
	userId := c.Request.Header.Get("userId")
	exists, err := fly.ImageExists(userId, project, task, db);
	if err != nil {
		log.Error(err)
		return
	} 

	if !exists {
		log.Debug("task does not exist")
		c.IndentedJSON(http.StatusNotFound, "Task " + task + " does not exist. Have you run `cakework deploy`?")
		return
	} 

	// serialize to json based on the type
	byteSlice, err := json.Marshal(runReq.Parameters)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Could not deserialize request params"})
		return
	}

	log.Debugf("Got a RunRequest: %+v\n", runReq)

	var req types.Run
	req.Task = task
	req.RunId = (uuid.New()).String()
	req.Project = project
	req.CPU = runReq.Compute.CPU
	req.Memory = runReq.Compute.Memory
	req.UserId = userId
	req.Status = "PENDING"

	req.Parameters = string(byteSlice)

	log.Debugf("Enqueueing request: %+v\n", req)
	if enqueue(req) != nil { // TODO check whether this is an err; if so, return different status code
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Sorry :( something broke, come talk to us"}) // TODO expose better errors
		return
	} else {
		c.IndentedJSON(http.StatusCreated, req)
		return
	}
}

func getUserFromHeader(c *gin.Context) (string, error) {
	clientToken := c.Request.Header.Get("X-Api-Key")
	userId, err := getUserFromAPIKey(clientToken)
	if err != nil {
		return "", err
	}
	return userId.Id, nil
}
