package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type task struct {
    ID     string  `json:"id"`
    Parameters  string  `json:"parameters"`
    Status string  `json:"status"`
    Result  string `json:"result"`
}


// album represents data about a record album.
type album struct {
    ID     string  `json:"id"`
    Title  string  `json:"title"`
    Artist string  `json:"artist"`
    Price  float64 `json:"price"`
}




type deployment struct {
    Name    string `json:"name"`
    Image     string  `json:"image"`
    User  string  `json:"user"`
    // Id  string `json:"id"`  // do they care about a deployment id? probably not. only an invocation id
    // Status  string `json:"status"`  // do they care about deployment status? 
    // Endpoint    string `json:"endpoint"` // maybe we shouldn't return this to the user and just store this in our db, so that when someone uses the client to invoke the activity they can do so
}

// albums slice to seed record album data.
var albums = []album{
    {ID: "3", Title: "Reputation", Artist: "Taylor Swift", Price: 56.99},
    {ID: "4", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
    {ID: "5", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

var db *sql.DB
var err error

func main() {
    // dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, database)
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
    router.GET("/albums", getAlbums)
    router.GET("/albums/:id", getAlbumByID)
    router.POST("/albums", postAlbums)
    router.POST("/deploy", deploy)
    
    router.POST("/call-task", callTask)
    // router.GET("/get-status", getStatus)
    // router.GET("/get-result", getResult)
    router.Run()
}

func deploy(c *gin.Context) {
    var newDeployment deployment
    if err := c.BindJSON(&newDeployment); err != nil {
        return
    }
}

// getAlbums responds with the list of all albums as JSON.
func getAlbums(c *gin.Context) {
    c.IndentedJSON(http.StatusOK, albums)
}

// postAlbums adds an album from JSON received in the request body.
func postAlbums(c *gin.Context) {
    var newAlbum album

    // Call BindJSON to bind the received JSON to
    // newAlbum.
    if err := c.BindJSON(&newAlbum); err != nil {
        return
    }

    // Add the new album to the slice.
    albums = append(albums, newAlbum)
    c.IndentedJSON(http.StatusCreated, newAlbum)
}

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
        log.Fatalf("impossible insert teacher: %s", err)
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
func getAlbumByID(c *gin.Context) {
    id := c.Param("id")

    // Loop through the list of albums, looking for
    // an album whose ID value matches the parameter.
    for _, a := range albums {
        if a.ID == id {
            c.IndentedJSON(http.StatusOK, a)
            return
        }
    }
    c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
}