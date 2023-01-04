package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	streamName     = "TASKS"
	streamSubjects = "TASKS.*"
	subjectName    = "TASKS.created"
)

type task struct {
    ID     string  `json:"id"`
    Parameters  string  `json:"parameters"`
    Status string  `json:"status"`
    Result  string `json:"result"`
}

type TaskRequest struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
	Task string `json:"task"`
    Parameters  string  `json:"parameters"`
}

// Q: how to share this type with the poller class?
type TaskRun struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
	Task string `json:"task"`
    Parameters  string  `json:"parameters"`
	RequestId     string  `json:"requestId"`
	Status	string `json:"status"`
	Result  string  `json:"result"`
}

type GetStatusRequest struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
    RequestId  string  `json:"requestId"`
}

type GetStatusResponse struct {
    Status     string  `json:"status"`
}

type GetResultRequest struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
    RequestId  string  `json:"requestId"`
}
// Q: how will errors be handled? TODO need to expose an error field?
type GetResultResponse struct {
    Result     string  `json:"result"`
}

type UpdateStatusRequest struct {
	UserId     string  `json:"userId"`
	App	string `json:"app"`
    RequestId  string  `json:"requestId"`
	Status  string  `json:"status"`
}

type UpdateResultRequest struct {
	UserId     string  `json:"userId"`
	App	string `json:"app"`
    RequestId  string  `json:"requestId"`
	Result  string  `json:"result"`
}

var db *sql.DB
var err error

var js nats.JetStreamContext
var local bool

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

	/*
	// Creates stream
	err = createStream(js)
	checkErr(err)
	// Create orders by publishing messages
	err = createOrder(js)
	checkErr(err)
	*/

    // // Simple Publisher
    // nc.Publish("foo", []byte("Hello World"))

    // // Simple Async Subscriber
    // nc.Subscribe("foo", func(m *nats.Msg) {
    //     fmt.Printf("Received a message: %s\n", string(m.Data))
    // })

    // nc.Publish("foo", []byte("Hello World 3"))

    dsn := "xodsymuvucvxj8a0fcvj:pscale_pw_wBKY0AVn5yilMTIVANcwmSxj2viJV76thiDTaNqHO96@tcp(us-west.connect.psdb.cloud)/sahale-application-db?tls=true"
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
    // router.GET("/albums", getAlbums)
    // router.GET("/albums/:id", getAlbumByID)
    // Q: what is 204 no content?
    router.POST("/submit-task", submitTask)
	// router.GET("/get-task", getTaskRun) //TODO we should probably have this
    router.GET("/get-status", getStatus) // TODO change to the syntax /status/:requestId? and /result/:requestId?
    router.GET("/get-result", getResult) 
    router.PATCH("/update-status", updateStatus)
	router.PATCH("/update-result", updateResult)
	router.Run()
}

// // getAlbums responds with the list of all albums as JSON.
// func getAlbums(c *gin.Context) {
//     c.IndentedJSON(http.StatusOK, albums)
// }

func submitTask(c *gin.Context) {
	// err = createOrder(js)

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

	newTaskRun := TaskRun {
		UserId: userId,
		App: app,
		Task: task,
		Parameters: newTaskRequest.Parameters,
		RequestId: (uuid.New()).String(),
		Status: "PENDING",
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

func getStatus(c *gin.Context) {
	// err = createOrder(js)

    var newGetStatusRequest GetStatusRequest

    if err := c.BindJSON(&newGetStatusRequest); err != nil {
		fmt.Println("got error reading in request")
		fmt.Println(err)
        return
    }

	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	// sanitize; convert app and task name to lower case, only hyphens
	userId := strings.Replace(strings.ToLower(newGetStatusRequest.UserId), "_", "-", -1)
	app := strings.Replace(strings.ToLower(newGetStatusRequest.App), "_", "-", -1)

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


// getAlbumByID locates the album whose ID value matches the id
// parameter sent by the client, then returns that album as a response.
// func getAlbumByID(c *gin.Context) {
//     id := c.Param("id")

//     // Loop through the list of albums, looking for
//     // an album whose ID value matches the parameter.
//     for _, a := range albums {
//         if a.ID == id {
//             c.IndentedJSON(http.StatusOK, a)
//             return
//         }
//     }
//     c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
// }

type Order struct {
	OrderID    int
	CustomerID string
	Status     string
}

// createOrder publishes stream of events
// with subject "ORDERS.created"
func createOrder(js nats.JetStreamContext) error {
	var order Order
	for i := 1; i <= 10; i++ {
		order = Order{
			OrderID:    i,
			CustomerID: "Cust-" + strconv.Itoa(i),
			Status:     "created",
		}
		orderJSON, _ := json.Marshal(order)
		_, err := js.Publish(subjectName, orderJSON)
		if err != nil {
			return err
		}
		log.Printf("Order with OrderID:%d has been published\n", i)
	}
	return nil
}

func createOneOrder(js nats.JetStreamContext) error {
	var order Order
	for i := 1; i <= 2; i++ {
		order = Order{
			OrderID:    i,
			CustomerID: "Cust-" + strconv.Itoa(i),
			Status:     "created",
		}
		orderJSON, _ := json.Marshal(order)
		_, err := js.Publish(subjectName, orderJSON)
		if err != nil {
			return err
		}
		log.Printf("Order with OrderID:%d has been published\n", i)
	}
	return nil
}

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
		fmt.Println("got task run: ")
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