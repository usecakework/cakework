package main

import (
	"context"
	"database/sql"
	"encoding/json"
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

var db *sql.DB
var err error

var js nats.JetStreamContext

func main() {

    nc, _ := nats.Connect(nats.DefaultURL)
    // nc, _ := nats.Connect("cakework-nats-cluster.internal")

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
    
    router.POST("/submit-task", submitTask)
    // router.GET("/get-status", getStatus)
    // router.GET("/get-result", getResult)
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
			log.Fatalf("impossible to insert : %s", err)
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