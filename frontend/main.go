package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

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

func main() {
    gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
    router.GET("/albums", getAlbums)
    router.GET("/albums/:id", getAlbumByID)
    router.POST("/albums", postAlbums)
    router.POST("/deploy", deploy)
    router.Run()
}

func deploy(c *gin.Context) {
    var newDeployment deployment
    if err := c.BindJSON(&newDeployment); err != nil {
        return
    }
    // check to see if app exists; if not, create one
    // naming scheme: userId-activity (or userId-workflow-activity)
    appName := newDeployment.User + "-" + newDeployment.Name
    // the create app command may fail. TODO handle the error. For now, just let it fail
    shell(exec.Command("fly", "apps", "create", "--name", appName, "--org", "sahale"))
    shell(exec.Command("fly", "machine", "run", newDeployment.Image, "--app", appName))
    // create a new fly machines app; make sure to namespace with name of user. for now, can just add a uuid?
    // start a new machine instance
    // TODO: insert new activity entry in the database
    c.IndentedJSON(http.StatusCreated, newDeployment) // q: return other parameters? like the invocation id?
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

// executes shell command, piping to stdout and stderr and to log file (TODO verify this)
func shell(cmd *exec.Cmd) {
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr
    err := cmd.Run()
    if err != nil {
        fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
        panic(err) // will this print it out in the right format?
    }
    fmt.Println("Result: " + out.String())
}