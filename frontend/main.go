package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
)

const (
	streamName     = "TASKS"
	streamSubjects = "TASKS.*"
	subjectName    = "TASKS.created"
)

type task struct {
	ID         string `json:"id"`
	Parameters string `json:"parameters"`
	Status     string `json:"status"`
	Result     string `json:"result"`
}

type TaskRequest struct {
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Parameters string `json:"parameters"`
}

// Q: how to share this type with the poller class?
type TaskRun struct {
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Parameters string `json:"parameters"`
	RequestId  string `json:"requestId"`
	Status     string `json:"status"`
	Result     string `json:"result"`
}

type GetStatusRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
}

type GetStatusResponse struct {
	Status string `json:"status"`
}

type GetResultRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
}

// Q: how will errors be handled? TODO need to expose an error field?
type GetResultResponse struct {
	Result string `json:"result"`
}

type UpdateStatusRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	Status    string `json:"status"`
}

type UpdateResultRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	Result    string `json:"result"`
}

type CreateClientTokenRequest struct {
	UserId string `json:"userId"` // Q: can we get the user id from the auth info?
	Name   string `json:"name"`
}

type ClientToken struct {
	Id     string `json:"id"`
	Token  string `json:"token"`
	UserId string `json:"userId"`
	Name   string `json:"name"`
}

type CreateUserRequest struct {
	UserId string `json:"userId"` // TODO: auto-generate a user id in our system?
}

type User struct {
	Id string `json:"id"`
}

type GetUserByClientTokenRequest struct {
	Token string `json:"token"`
}

type GetUserRequest struct {
	UserId string `json:"userId"`
}

type GetTaskStatusRequest struct {
	UserId string `json:"userId"`
	App    string `json:"app"`
	Task   string `json:"task"`
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
	localPtr := flag.Bool("local", false, "boolean which if true runs the poller locally") // can pass go run main.go -local
	flag.Parse()

	local = *localPtr

	var nc *nats.Conn
	if local == true {
		nc, _ = nats.Connect(nats.DefaultURL)
		fmt.Println("Local mode; connected to nats cluster: " + nats.DefaultURL)
	} else {
		nc, _ = nats.Connect("cakework-nats-cluster.internal")
		fmt.Println("Non-local mode; connected to nats cluster: cakework-nats-cluster.internal")
	}

	// Creates JetStreamContext
	js, err = nc.JetStream()
	checkErr(err)

	// Creates stream - note: should we need to do this every time? will this cause old stuff to be lost
	// Q: durability of the stream? where to store it? if reboot will things get lost?
	err = createStream(js)
	checkErr(err)

	dsn := "o8gbhwxuuk6wktip1q0x:pscale_pw_2UIlU6gaoTm7UBXYCbWCuHCkFYqO5pkJQmSri74KRn5@tcp(us-west.connect.psdb.cloud)/cakework?tls=true&parseTime=true"
	// Open the connection
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("impossible to create the connection: %s", err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())
	router.Use(guidMiddleware())

	// The issuer of our token.
	issuerURL, _ := url.Parse("https://dev-qanxtedlpguucmz5.us.auth0.com/")

	// The audience of our token.
	audience := "https://cakework-frontend.fly.dev"

	provider := jwks.NewCachingProvider(issuerURL, time.Duration(5*time.Minute))

	jwtValidator, _ := validator.New(provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{audience},
		validator.WithCustomClaims(customClaims),
	)

	jwtMiddleware := jwtmiddleware.New(jwtValidator.ValidateToken)

	router.Use(adapter.Wrap(jwtMiddleware.CheckJWT))

	router.POST("/submit-task", submitTask)
	// router.GET("/get-task", getTaskRun) //TODO we should probably have this
	router.GET("/get-status", getStatus)                              // TODO change to GET /request/status/requestId
	router.GET("/get-result", getResult)                              // TODO change to GET /request/result/requestId
	router.PATCH("/update-status", updateStatus)                      // TODO change to POST /request/status/requestId
	router.PATCH("/update-result", updateResult)                      // TODO change to POST /request/result/requestId
	router.POST("/create-client-token", createClientToken)            // TODO change to POST /client-token // TODO protect this using auth0
	router.POST("/create-user", createUser)                           // TODO change to POST /user
	router.GET("/get-user-from-client-token", getUserFromClientToken) // TODO change to GET /user with parameters/query string
	router.GET("/get-user", getUser)                                  // TODO change to GET /user
	router.GET("/task/logs", handleGetTaskLogs)
	// TODO have an add-task
	router.Run()
}

func guidMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.New()
		c.Set("uuid", uuid)
		fmt.Printf("The request with uuid %s is started \n", uuid)
		c.Next()
		fmt.Printf("The request with uuid %s is served \n", uuid)
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

func submitTask(c *gin.Context) {
	if !isAuthed(c, "submit:task") {
		return
	}

	var newTaskRequest TaskRequest

	if err := c.BindJSON(&newTaskRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newTaskRequest.UserId), "_", "-", -1)
	app := strings.Replace(strings.ToLower(newTaskRequest.App), "_", "-", -1)
	task := strings.Replace(strings.ToLower(newTaskRequest.Task), "_", "-", -1)

	newTaskRun := TaskRun{
		UserId:     userId,
		App:        app,
		Task:       task,
		Parameters: newTaskRequest.Parameters,
		RequestId:  (uuid.New()).String(),
		Status:     "PENDING",
	}

	// enqueue this message
	if createTaskRun(newTaskRun) != nil { // TODO check whether this is an err; if so, return different status code
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
	} else {
		c.IndentedJSON(http.StatusCreated, newTaskRun)
	}
	// store this into the database

	// query := "INSERT INTO `Request2` (`id`, `status`, `parameters`) VALUES (?, ?, ?)"
	// insertResult, err := db.ExecContext(context.Background(), query, newTaskRequest.ID, newTaskRequest.Status, newTaskRequest.Parameters)
	// if err != nil {
	//     log.Fatalf("impossible to insert : %s", err)
	// }
	// id, err := insertResult.LastInsertId()
	// if err != nil {
	//     log.Fatalf("impossible to retrieve last inserted id: %s", err)
	// }
	// log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?

	// TODO enqueue the task into NATS

	// ok that for post that we return something different?

}

func handleGetTaskLogs(c *gin.Context) {
	if !isAuthed(c, "get:task_status") {
		return
	}

	var newGetTaskStatusRequest GetTaskStatusRequest

	if err := c.BindJSON(&newGetTaskStatusRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	userId := sanitizeUserId(newGetTaskStatusRequest.UserId)
	app := sanitizeAppName(newGetTaskStatusRequest.App)
	task := sanitizeTaskName(newGetTaskStatusRequest.Task)

	taskStatus, err := getTaskStatus(db, userId, app, task)

	// return task not found properly

	if err != nil {
		fmt.Println("Error when getting task logs")
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "sorry :( something broke, come talk to us"})
	}

	c.IndentedJSON(http.StatusOK, taskStatus)
}

func getStatus(c *gin.Context) {
	if !isAuthed(c, "get:status") {
		return
	}

	var newGetStatusRequest GetStatusRequest

	if err := c.BindJSON(&newGetStatusRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := sanitizeUserId(newGetStatusRequest.UserId)
	app := sanitizeAppName(newGetStatusRequest.App)

	taskRun, err := getTaskRun(userId, app, newGetStatusRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetStatusRequest.RequestId + " not found"})
		} else {
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		}
	} else {
		response := GetStatusResponse{
			Status: taskRun.Status,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func getResult(c *gin.Context) {
	if !isAuthed(c, "get:result") {
		return
	}

	var newGetResultRequest GetResultRequest

	if err := c.BindJSON(&newGetResultRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newGetResultRequest.UserId), "_", "-", -1)
	app := strings.Replace(strings.ToLower(newGetResultRequest.App), "_", "-", -1)

	taskRun, err := getTaskRun(userId, app, newGetResultRequest.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no request with request id found")
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "request with request id " + newGetResultRequest.RequestId + " not found"})
		} else {
			c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO expose better errors
		}
	} else {
		response := GetResultResponse{
			Result: taskRun.Result,
		}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func updateStatus(c *gin.Context) {
	if !isAuthed(c, "update:status") {
		return
	}

	var request UpdateStatusRequest

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
	if !isAuthed(c, "update:result") {
		return
	}

	var request UpdateResultRequest

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

// no need to add scope checking here, as this is not directly invoked by a route
func createTaskRun(taskRun TaskRun) error {
	taskRunJSON, _ := json.Marshal(taskRun)

	_, err := js.Publish(subjectName, taskRunJSON)
	if err != nil {
		fmt.Println("error while publishing")
		fmt.Println(err)
		return err // TODO return a human readable error
	} else {
		log.Printf("Task run with RequestId:%s has been published\n", taskRun.RequestId)
		// update the database
		log.Printf("Inserting into db now")
		query := "INSERT INTO `TaskRun` (`requestId`, `userId`, `app`, `task`, `parameters`, `status`) VALUES (?, ?, ?, ?, ?, ?)"
		insertResult, err := db.ExecContext(context.Background(), query, taskRun.RequestId, taskRun.UserId, taskRun.App, taskRun.Task, taskRun.Parameters, taskRun.Status)
		if err != nil {
			fmt.Printf("impossible to insert : %s", err)
			return err
		}
		id, err := insertResult.LastInsertId()
		if err != nil {
			return err
			// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
		} else {
			log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?
			log.Printf("Successfully inserted")
		}
	}
	return nil
}

func getTaskRun(userId string, app string, requestId string) (TaskRun, error) {

	// TODO use the userId and app
	var taskRun TaskRun
	var result sql.NullString
	err = db.QueryRow("SELECT userId, app, task, parameters, requestId, status, result FROM TaskRun where requestId = ?", requestId).Scan(&taskRun.UserId, &taskRun.App, &taskRun.Task, &taskRun.Parameters, &taskRun.RequestId, &taskRun.Status, &result)
	if err != nil {
		// if err == sql.ErrNoRows {
		return taskRun, err
		// }
		// log.Fatalf("impossible to fetch : %s", err) // we shouldn't exit??? or will this only kill the current thing? TODO test this behavior
	} else {
		if result.Valid {
			taskRun.Result = result.String
		}
		fmt.Println(taskRun)
		return taskRun, nil
	}

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
	if !isAuthed(c, "create:client_token") {
		return
	}

	var newRequest CreateClientTokenRequest
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

	clientToken := ClientToken{
		Id:     tokenId,
		Token:  token,
		UserId: newRequest.UserId,
		Name:   newRequest.Name,
	}

	query := "INSERT INTO `ClientToken` (`id`, `name`, `token`, `userId`) VALUES (?, ?, ?, ?)"
	insertResult, err := db.ExecContext(context.Background(), query, clientToken.Id, clientToken.Name, clientToken.Token, clientToken.UserId)
	if err != nil {
		fmt.Printf("impossible to insert : %s", err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	id, err := insertResult.LastInsertId()
	if err != nil {
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?
		log.Printf("Successfully inserted")
		fmt.Println(clientToken)
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
	if !isAuthed(c, "create:user") {
		return
	}

	var newRequest CreateUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newRequest.UserId), "_", "-", -1)

	newUser := User{
		Id: userId,
	}

	query := "INSERT INTO `User` (`id`) VALUES (?)"
	insertResult, err := db.ExecContext(context.Background(), query, newUser.Id)
	if err != nil {
		fmt.Printf("impossible to insert : %s", err)
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
	}
	id, err := insertResult.LastInsertId()
	if err != nil {
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"})
		// log.Fatalf("impossible to retrieve last inserted id: %s", err) // will this cause an error exit?
	} else {
		log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?
		log.Printf("Successfully inserted")
		c.IndentedJSON(http.StatusCreated, newUser)
	}
}

func getUserFromClientToken(c *gin.Context) {
	if !isAuthed(c, "get:user_from_client_token") {
		return
	}

	// fetch the client token by the token value
	// return the user
	var newRequest GetUserByClientTokenRequest

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	var user User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", newRequest.Token).Scan(&user.Id)
	if err != nil {
		// if err == sql.ErrNoRows {
		c.IndentedJSON(http.StatusFailedDependency, gin.H{"message": "internal server error"}) // TODO check for if got no rows
		// }
		// log.Fatalf("impossible to fetch : %s", err) // we shouldn't exit??? or will this only kill the current thing? TODO test this behavior
	} else {
		c.IndentedJSON(http.StatusOK, user)
	}
}

func getUser(c *gin.Context) {
	if !isAuthed(c, "get:user") {
		return
	}

	var newRequest GetUserRequest

	if err := c.BindJSON(&newRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
		return
	}

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newRequest.UserId), "_", "-", -1)

	// TODO use the userId and app
	var user User
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

func isAuthed(c *gin.Context, scope string) bool {
	claims, ok := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !ok {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"message": "Failed to get validated JWT claims."},
		)
		return false
	}

	customClaims, ok := claims.CustomClaims.(*CustomClaimsExample)
	if !ok {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"message": "Failed to cast custom JWT claims to specific type."},
		)
		return false
	}

	if len(customClaims.Scope) == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			map[string]string{"message": "Scope in JWT claims was empty."},
		)
		return false
	}

	if !strings.Contains(customClaims.Scope, scope) {
		c.IndentedJSON(http.StatusForbidden, `{"message":"Insufficient scope."}`)
		return false
	}

	return true
}
