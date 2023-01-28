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
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	cwHttp "github.com/usecakework/cakework/lib/http"
	"github.com/usecakework/cakework/lib/types"
	"github.com/usecakework/cakework/lib/util"
)

const (
	streamName     = "TASKS"
	streamSubjects = "TASKS.*"
	subjectName    = "TASKS.created"
)

// deprecated?
type task struct {
	ID         string `json:"id"`
	Parameters string `json:"parameters"`
	Status     string `json:"status"`
	Result     string `json:"result"`
}

// deprecated?
type TaskRequest struct {
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Parameters string `json:"parameters"`
}

var db *sql.DB
var err error

var js nats.JetStreamContext
var local bool

// CustomClaimsExample contains custom data we want from the token.
type CustomClaimsExample struct {
	Scope string `json:"scope"`
}

// Validate errors out if `ShouldReject` is true.
func (c *CustomClaimsExample) Validate(ctx context.Context) error {
	// if c.ShouldReject {
	// 	return errors.New("should reject was set to true")
	// }
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
	verbosePtr := flag.Bool("verbose", false, "boolean which if true runs the poller locally") // can pass go run main.go -local

	flag.Parse()

	verbose := *verbosePtr

	if verbose {
		log.Info("Verbose=true")
		log.SetLevel(log.DebugLevel)
	} else {
		log.Info("Verbose=false")
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
		log.Fatalf("impossible to create the connection: %s", err)
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
		jwtProtectedGroup.POST("/submit-task", submitTask, jwtTokenMiddleware("submit:task")) // the scope middleware needs to run before the jwtTokenMiddleware handler
		jwtProtectedGroup.GET("/get-status", getStatus, jwtTokenMiddleware("get:status"))
		jwtProtectedGroup.GET("/get-result", getResult, jwtTokenMiddleware("get:result"))                                              // TODO change to GET /request/result/requestId
		jwtProtectedGroup.PATCH("/update-status", updateStatus, jwtTokenMiddleware("update:status"))                                   // TODO change to POST /request/status/requestId
		jwtProtectedGroup.PATCH("/update-result", updateResult, jwtTokenMiddleware("update:result"))                                   // TODO change to POST /request/result/requestId
		jwtProtectedGroup.POST("/create-client-token", createClientToken, jwtTokenMiddleware("create:client_token"))                   // TODO change to POST /client-token // TODO protect this using auth0
		jwtProtectedGroup.POST("/create-user", createUser, jwtTokenMiddleware("create:user"))                                          // TODO change to POST /user
		jwtProtectedGroup.GET("/get-user-from-client-token", getUserFromClientToken, jwtTokenMiddleware("get:user_from_client_token")) // TODO change to GET /user with parameters/query string
		jwtProtectedGroup.GET("/get-user", getUser, jwtTokenMiddleware("get:user"))                                                    // TODO change to GET /user
		jwtProtectedGroup.GET("/task/logs", handleGetTaskLogs, jwtTokenMiddleware("get:task_status"))                                  // the scope is incorrectly named. TODO fix
		jwtProtectedGroup.GET("/request/logs", handleGetRequestLogs)
		jwtProtectedGroup.POST("/create-machine", createMachine, jwtTokenMiddleware("create:machine"))
		jwtProtectedGroup.PATCH("/update-machine-id", updateMachineId, jwtTokenMiddleware("update:machine_id")) // TODO change to POST /request/status/requestId
		jwtProtectedGroup.GET("/get-cli-secrets", getCLISecrets, jwtTokenMiddleware("create:user")) // wrong scope
		// TODO have an add-task
	}

	apiKeyProtectedGroup := router.Group("/client", apiKeyMiddleware())
	{
		apiKeyProtectedGroup.GET("/get-status", getStatus)
		apiKeyProtectedGroup.GET("/get-result", getResult) // TODO change to GET /request/result/requestId
		apiKeyProtectedGroup.POST("/submit-task", submitTask)
		apiKeyProtectedGroup.GET("/get-user-from-client-token", getUserFromClientToken) // user never actually invokes this, but our client library needs to
		apiKeyProtectedGroup.PATCH("/update-status", updateStatus)                                   // TODO change to POST /request/status/requestId
		apiKeyProtectedGroup.PATCH("/update-result", updateResult)                                   // TODO change to POST /request/result/requestId
	
		apiKeyProtectedGroup.GET("/runs/:runId/status", getRunStatus) // TODO migrate to getStatus
		apiKeyProtectedGroup.GET("/runs/:runId/result", getRunResult)
		apiKeyProtectedGroup.POST("/runs/", run)
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

// TODO have this throw error?
func submitTask(c *gin.Context) {
	// TODO check if app exists; if not, throw an error
	var req types.Request

	if err := c.BindJSON(&req); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(req.UserId), "_", "-", -1)
	app := strings.Replace(strings.ToLower(req.App), "_", "-", -1)
	task := strings.Replace(strings.ToLower(req.Task), "_", "-", -1)

	req.UserId = userId
	req.App = app
	req.Task = task
	req.RequestId = (uuid.New()).String()
	req.Status = "PENDING"

	// enqueue this message
	if enqueue(req) != nil { // TODO check whether this is an err; if so, return different status code
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "Internal server error"}) // TODO expose better errors
	} else {
		c.IndentedJSON(http.StatusCreated, req)
	}
}

func handleGetRequestLogs(c *gin.Context) {

	// get app name and task name from the request id
	var newGetRequestLogsRequest types.GetRequestLogsRequest

	if err := c.BindJSON(&newGetRequestLogsRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		c.IndentedJSON(http.StatusBadRequest, "Issue with parsing request to json")
		return
	}

	requestDetails, err := getRun(db, newGetRequestLogsRequest.RequestId)

	if err != nil {
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, "sorry :( something broke, come talk to us")
		return
	}

	if requestDetails == nil {
		c.IndentedJSON(http.StatusNotFound, "Request "+newGetRequestLogsRequest.RequestId+" does not exist.")
		return
	}

	requestId := requestDetails.RequestId
	userId := requestDetails.UserId
	appName := requestDetails.App
	taskName := requestDetails.Task
	machineId := requestDetails.MachineId

	logs, err := getRequestLogs(userId, appName, taskName, machineId, requestId)
	if err != nil {
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, "sorry :( something broke, come talk to us")
		return
	}

	c.IndentedJSON(http.StatusOK, logs)
	return
}

func handleGetTaskLogs(c *gin.Context) {
	var newGetTaskLogsRequest types.GetTaskLogsRequest

	if err := c.BindJSON(&newGetTaskLogsRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		c.IndentedJSON(http.StatusBadRequest, "Issue with parsing request to json")
		return
	}

	userId := util.SanitizeUserId(newGetTaskLogsRequest.UserId)
	app := util.SanitizeAppName(newGetTaskLogsRequest.App)
	task := util.SanitizeTaskName(newGetTaskLogsRequest.Task)
	statusFilter := newGetTaskLogsRequest.StatusFilter

	taskLogs, err := GetTaskLogs(db, userId, app, task, statusFilter)

	// return task not found properly
	if err != nil {
		fmt.Println("Error when getting task logs")
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, "sorry :( something broke, come talk to us")
		return
	}

	c.IndentedJSON(http.StatusOK, taskLogs)
}

func getStatus(c *gin.Context) {
	var newGetStatusRequest types.GetStatusRequest

	if err := c.BindJSON(&newGetStatusRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	request, err := getRun(db, newGetStatusRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetStatusRequest.RequestId + " not found"})
			return
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "sorry :( something broke, come talk to us"}) // TODO expose better errors
			return
		}
	} else {
		response := types.GetStatusResponse{
			Status: request.Status,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func getCLISecrets(c *gin.Context) {
	FLY_ACCESS_TOKEN := viper.GetString("FLY_ACCESS_TOKEN")
	if FLY_ACCESS_TOKEN == "" {
		log.Info("FLY_ACCESS_TOKEN shouldn't be null")
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "FLY_ACCESS_TOKEN shouldn't be null"})
		return
	}

	secrets := types.CLISecrets {
		FLY_ACCESS_TOKEN: FLY_ACCESS_TOKEN,
	}

	c.IndentedJSON(http.StatusOK, secrets)
}

func getResult(c *gin.Context) {
	var newGetResultRequest types.GetResultRequest

	if err := c.BindJSON(&newGetResultRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	request, err := getRun(db, newGetResultRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetResultRequest.RequestId + " not found"})
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "sorry :( something broke, come talk to us"}) // TODO expose better errors
		}
	} else {
		response := types.GetResultResponse{
			Result: request.Result,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func updateStatus(c *gin.Context) {
	var request types.UpdateStatusRequest

	if err := c.BindJSON(&request); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO verify that we aren't overwriting anything
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// userId := strings.Replace(strings.ToLower(request.UserId), "_", "-", -1)
	// app := strings.Replace(strings.ToLower(request.App), "_", "-", -1)

	// TODO use the userId and app
	stmt, err := db.Prepare("UPDATE TaskRun SET status = ? WHERE requestId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.Status, request.RequestId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		fmt.Println("nothing was updated")
		c.Status(http.StatusNotFound)
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func updateResult(c *gin.Context) {
	var request types.UpdateResultRequest

	if err := c.BindJSON(&request); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO verify that we aren't overwriting anything
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// userId := strings.Replace(strings.ToLower(request.UserId), "_", "-", -1)
	// app := strings.Replace(strings.ToLower(request.App), "_", "-", -1)

	// TODO use the userId and app
	stmt, err := db.Prepare("UPDATE TaskRun SET result = ? WHERE requestId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.Result, request.RequestId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		fmt.Println("nothing was updated")
		c.Status(http.StatusNotFound)
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		} else {
			c.Status(http.StatusOK)
		}
	}
}

// right now just updating the TaskRun table; eventually migrate to Request table
func updateMachineId(c *gin.Context) {
	var request types.UpdateMachineId

	if err := c.BindJSON(&request); err != nil {
		fmt.Println("got error reading in request")
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"}) // TODO expose better errors
	}

	// TODO verify that we aren't overwriting anything
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// userId := strings.Replace(strings.ToLower(request.UserId), "_", "-", -1)
	// app := strings.Replace(strings.ToLower(request.App), "_", "-", -1)

	// TODO use the userId and app
	stmt, err := db.Prepare("UPDATE TaskRun SET machineId = ? WHERE requestId = ?")
	checkErr(err)

	res, e := stmt.Exec(request.MachineId, request.RequestId)
	checkErr(e)

	a, e := res.RowsAffected()
	checkErr(e)
	fmt.Printf("Updated %d rows", a)
	if a == 0 {
		// nothing was updated; row not found most likely (though can be due to some other error)
		fmt.Println("nothing was updated")
		c.Status(http.StatusNotFound)
	} else {
		if err != nil {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"}) // TODO expose better errors
		} else {
			c.Status(http.StatusOK)
		}
	}
}

// no need to add scope checking here, as this is not directly invoked by a route
func enqueue(req types.Request) error {
	reqJSON, _ := json.Marshal(req)

	_, err := js.Publish(subjectName, reqJSON)
	if err != nil {
		fmt.Println("error while publishing")
		fmt.Println(err)
		return err // TODO return a human readable error
	} else {
		log.Printf("Request with RequestId:%s has been published\n", req.RequestId)
		// update the database
		log.Printf("Inserting into db now")
		query := "INSERT INTO `TaskRun` (`requestId`, `userId`, `app`, `task`, `parameters`, `status`, `cpu`, `memoryMB`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		insertResult, err := db.ExecContext(context.Background(), query, req.RequestId, req.UserId, req.App, req.Task, req.Parameters, req.Status, req.CPU, req.MemoryMB)
		if err != nil {
			fmt.Printf("impossible to insert : %s", err)
			return err
		}
		_, err = insertResult.LastInsertId()
		if err != nil {
			return err
			// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
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
func createClientToken(c *gin.Context) {
	fmt.Println("context")
	fmt.Println(c)
	var newRequest types.CreateClientTokenRequest
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Println("Failed to generate random token")
		fmt.Println(err) // TODO return a custom http response
	}

	token := hex.EncodeToString(b)

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
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
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		c.IndentedJSON(http.StatusCreated, clientToken)
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// userId := strings.Replace(strings.ToLower(newRequest.UserId), "_", "-", -1)

	// generate 32 character token

	// TODO: insert the token into the database
	// TODO handle error if can't create token

}

func createUser(c *gin.Context) {
	var newRequest types.CreateUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newRequest.UserId), "_", "-", -1)

	newUser := types.User{
		Id: userId,
	}

	query := "INSERT INTO `User` (`id`) VALUES (?)"
	insertResult, err := db.ExecContext(context.Background(), query, newUser.Id)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		// log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?
		log.Printf("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, newUser)
	}
}

func getUserFromClientToken(c *gin.Context) {

	// fetch the client token by the token value
	// return the user
	var newRequest types.GetUserByClientTokenRequest

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	var user types.User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", newRequest.Token).Scan(&user.Id)
	if err != nil {
		log.Error(err)
		if err.Error() == sql.ErrNoRows.Error() {
			c.IndentedJSON(http.StatusBadRequest, "Please provide a valid client token.")
		} else {
			log.Debug("TODO catch the specific error (such as access denied)")
			c.IndentedJSON(http.StatusInternalServerError, "Something went wrong :( Please contact us.")
		}
	} else {
		c.IndentedJSON(http.StatusOK, user)
	}
}

func getUser(c *gin.Context) {
	var newRequest types.GetUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newRequest.UserId), "_", "-", -1)

	// TODO use the userId and app
	var user types.User
	err = db.QueryRow("SELECT id FROM User where id = ?", userId).Scan(&user.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "user with id not found"})
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		}
		// log.Fatalf("impossible to fetch : %s", err) // we shouldn't exit??? or will this only kill the current thing? TODO test this behavior
	} else {
		fmt.Println("user")
		fmt.Println(user)
		c.IndentedJSON(http.StatusOK, user)
	}
}

// TODO put into a separate package. Can have the main.go invoke this as well
func getUserFromAPIKey(apiKey string) (*types.User, error) {
	// fetch the client token by the token value
	// return the user
	newRequest := types.GetUserByClientTokenRequest{
		Token: apiKey,
	}
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	var user types.User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", newRequest.Token).Scan(&user.Id)
	if err != nil && user.Id != "" {
		return nil, err
	} else {
		return &user, nil
	}
}

func createMachine(c *gin.Context) {
	var req types.CreateMachineRequest

	if err := c.BindJSON(&req); err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	// TODO put this into a middleware
	userId := util.SanitizeUserId(req.UserId)
	project := util.SanitizeProjectName(req.Project)
	task := util.SanitizeTaskName(req.Task)
	flyApp := userId + "-" + project + "-" + task

	query := "INSERT INTO `FlyMachine` (`userId`, `project`, `task`, `flyApp`, `name`, `machineId`, `state`, `image`, `source`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertResult, err := db.ExecContext(context.Background(), query, userId, project, task, flyApp, req.Name, req.MachineId, req.State, req.Image, req.Source)
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	} else {
		log.Info("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, req)
	}
}

// new refactor using REST API patterns
// TODO take i the project id
func getRunStatus(c *gin.Context) {
	runId := c.Param("runId") 
	// get project and user id from the headers
	// TODO cache the info about the user id 
	
	// apiKey := c.Request.Header.Get("X-Api-Key") // TODO refactor to middleware
	// userId, err := getUserFromAPIKey(apiKey)
	// if err != nil {
	// 	c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
	// 	return
	// }

	// TODO add userId and project id to the requestDetails call

	// project := c.Request.Header.Get("project")

	request, err := getRun(db, runId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + runId + " not found"})
			return	
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"}) // TODO expose better errors
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
func getRunResult(c *gin.Context) {
	runId := c.Param("runId")

	request, err := getRun(db, runId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Run with run id " + runId + " not found"})
			return	
		} else {
			log.Error(err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"}) // TODO expose better errors
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
func run(c *gin.Context) {
	// TODO check if app exists; if not, throw an error
	var runReq types.RunRequest
	// get user id and project from the headers
	clientToken := c.Request.Header.Get("X-Api-Key")
	userId, err := getUserFromAPIKey(clientToken)
	if err != nil {
		log.Error("Error getting user id from client token")
		log.Error(err)
		c.IndentedJSON(http.StatusInternalServerError, "sorry :( something broke, come talk to us")
	}

	log.Debug("user id: " + userId.Id)

	if err := c.BindJSON(&runReq); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	project := util.SanitizeAppName(c.Request.Header.Get("name"))
	// userId := "dummy"
	// app := "myapp"
	task := util.SanitizeTaskName(runReq.Task)
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?
	// TODO hardcode stuff for now!! 
	// sanitize; convert app and task name to lower case, only hyphens
	// userId := strings.Replace(strings.ToLower(req.UserId), "_", "-", -1)
	// app := strings.Replace(strings.ToLower(req.App), "_", "-", -1)
	// task := strings.Replace(strings.ToLower(req.Task), "_", "-", -1)

	// req.UserId = userId
	// req.App = app
	// log.Debug(runReq.Parameters)
	// for _, value := range runReq.Parameters {
	// 	log.Debug(value)
	// 	fmt.Printf("t1: %T\n", value	)
	//   }

	var req types.Request
	req.Task = task
	req.RequestId = (uuid.New()).String()
	req.App = project
	req.CPU = runReq.CPU
	req.MemoryMB = runReq.Memory
	req.UserId = userId.Id
	req.Status = "PENDING"
	
	// serialize to json based on the type
	byteSlice, _ := json.Marshal(runReq.Parameters)
	req.Parameters = string(byteSlice)


	log.Debugf("%+v\n", req)
	if enqueue(req) != nil { // TODO check whether this is an err; if so, return different status code
		log.Error(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		return
	} else {
		c.IndentedJSON(http.StatusCreated, req)
		return
	}
}
