package main

import (
	"io"
	"log"
	"net/http"
	"os"
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

// albums slice to seed record album data.
var albums = []album{
    {ID: "3", Title: "Reputation", Artist: "Taylor Swift", Price: 56.99},
    {ID: "4", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
    {ID: "5", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

func main() {
    deploy("mongo-express")
    // shell(exec.Command("fly", "machines", "api-proxy", "--org", "sahale", "&"))

    gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
    router.GET("/albums", getAlbums)
    router.GET("/albums/:id", getAlbumByID)
    router.POST("/albums", postAlbums)

    router.Run()
}

func deploy(image string) {
    shell(exec.Command("fly", "machine", "run", image, "--app", "jessie-activity-test"))
    // create a new fly machines app; make sure to namespace with name of user. for now, can just add a uuid?
    // start a new machine instance

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
    f, err := os.OpenFile("sahalectl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("Error opening file: %v", err)
    }
    defer f.Close()

   
    mwriter := io.MultiWriter(f, os.Stdout)
    cmd.Stderr = mwriter
    cmd.Stdout = mwriter
    err = cmd.Run() //blocks until sub process is complete
    if err != nil {
        panic(err)
    }
}