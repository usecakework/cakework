package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
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

func main() {
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	flag.Parse()

	local = *localPtr

	var nc *nats.Conn
	if local == true {
		nc, _ = nats.Connect(nats.DefaultURL)
		fmt.Println("Local mode; connected to nats cluster: " + nats.DefaultURL)
	} else {
		NATS_CLUSTER := viper.GetString("NATS_CLUSTER")
		nc, _ = nats.Connect(NATS_CLUSTER)
		fmt.Println("Non-local mode; connected to nats cluster: " + NATS_CLUSTER)
	}

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
	router.Use(guidMiddleware())

	// The issuer of our token.
	AUTH0_URL := viper.GetString("AUTH0_URL")
	issuerURL, _ := url.Parse(AUTH0_URL)

	// The audience of our token.
	FRONTEND_URL := viper.GetString("FRONTEND_URL")
	audience := FRONTEND_URL

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

		// TODO have an add-task
	}

	apiKeyProtectedGroup := router.Group("/client", apiKeyMiddleware())
	{
		apiKeyProtectedGroup.GET("/get-status", getStatus)
		apiKeyProtectedGroup.GET("/get-result", getResult) // TODO change to GET /request/result/requestId
		apiKeyProtectedGroup.POST("/submit-task", submitTask)
		apiKeyProtectedGroup.GET("/get-user-from-client-token", getUserFromClientToken) // user never actually invokes this, but our client library needs to
	}

	router.Run()
}

func guidMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.New()
		c.Set("uuid", uuid)
		log.Printf("Request started: %s\n", uuid)
		reqDump, err := httputil.DumpRequest(c.Request, true) // TODO delete this later!
		if err != nil {
			log.Error("Got error while printing request")
			log.Error(err)
		} else {
			log.Info(string(reqDump)) // TODO delete
		}
		c.Next()
		log.Printf("Request finished: %s\n", uuid)
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.New()
		c.Set("uuid", uuid)
		fmt.Printf("The request with uuid %s is started \n", uuid)
		c.Next()
		fmt.Printf("The request with uuid %s is served \n", uuid)
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
	if createRequest(req) != nil { // TODO check whether this is an err; if so, return different status code
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
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

	requestDetails, err := getRequestDetails(db, newGetRequestLogsRequest.RequestId)

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

	logs, err := getRequestLogs(userId, appName, taskName, requestId)
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

	request, err := getRequestDetails(db, newGetStatusRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetStatusRequest.RequestId + " not found"})
		} else {
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		}
	} else {
		response := types.GetStatusResponse{
			Status: request.Status,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func getResult(c *gin.Context) {
	var newGetResultRequest types.GetResultRequest

	if err := c.BindJSON(&newGetResultRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	request, err := getRequestDetails(db, newGetResultRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetResultRequest.RequestId + " not found"})
		} else {
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
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
		fmt.Println(err)
		return
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
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		} else {
			c.Status(http.StatusOK)
		}
	}
}

// no need to add scope checking here, as this is not directly invoked by a route
func createRequest(req types.Request) error {
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
		fmt.Printf("impossible to insert : %s", err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
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
		fmt.Println("got error reading in request")
		fmt.Println(err)
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
		fmt.Printf("impossible to insert : %s", err)
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
		if err.Error() == sql.ErrNoRows.Error() {
			c.IndentedJSON(http.StatusBadRequest, "Invalid client token.")
		} else {
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
		fmt.Println("got error reading in request")
		fmt.Println(err)
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
		log.Printf("impossible to insert : %s", err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	_, err = insertResult.LastInsertId()
	if err != nil {
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		log.Error("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		log.Info("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, req)
	}
}
