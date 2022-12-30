package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"

	"proto/cakework"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	streamName     = "ORDERS"
	streamSubjects = "ORDERS.*"
	subjectName    = "ORDERS.created"
)

type task struct {
    ID     string  `json:"id"`
    Parameters  string  `json:"parameters"`
    Status string  `json:"status"`
    Result  string `json:"result"`
}

var db *sql.DB
var err error

func main() {
    ////// testing grpc go client
	var conn *grpc.ClientConn
	// conn, err := grpc.Dial(":9000", grpc.WithInsecure())
    conn, err := grpc.Dial("shared-app-say-hello.fly.dev", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := cakework.NewCakeworkClient(conn)

	createReq := cakework.Request{Parameters: "{\"name\": \"jessie\""}
	response, err := c.Create(context.Background(), &createReq)
	if err != nil {
		log.Fatalf("Error Cakework RunActivity: %s", err)
	}
	log.Printf("Response from server: %s", response.Message)

    panic("no disco")





    //////


    // nc, _ := nats.Connect(nats.DefaultURL)
    nc, _ := nats.Connect("cakework-nats-cluster.internal")

	// Creates JetStreamContext
	js, err := nc.JetStream()
	checkErr(err)
	// Creates stream
	err = createStream(js)
	checkErr(err)
	// Create orders by publishing messages
	err = createOrder(js)
	checkErr(err)


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
    
    router.POST("/call-task", callTask)
    // router.GET("/get-status", getStatus)
    // router.GET("/get-result", getResult)
    router.Run()
}

// // getAlbums responds with the list of all albums as JSON.
// func getAlbums(c *gin.Context) {
//     c.IndentedJSON(http.StatusOK, albums)
// }

func callTask(c *gin.Context) {
    var newTask task

    if err := c.BindJSON(&newTask); err != nil {
        return
    }

    newTask.ID = (uuid.New()).String()
    newTask.Status = "PENDING"

    query := "INSERT INTO `Request2` (`id`, `status`, `parameters`) VALUES (?, ?, ?)"
    insertResult, err := db.ExecContext(context.Background(), query, newTask.ID, newTask.Status, newTask.Parameters)
    if err != nil {
        log.Fatalf("impossible to insert : %s", err)
    }
    id, err := insertResult.LastInsertId()
    if err != nil {
        log.Fatalf("impossible to retrieve last inserted id: %s", err)
    }
    log.Printf("inserted id: %d", id) // TODO this is not working as expected? or should this always return 0? should we turn on auto-increment?

    // TODO enqueue the task into NATS


    c.IndentedJSON(http.StatusCreated, newTask)
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