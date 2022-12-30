package main

// Q: should we just expose a gin gonic server with no methods?
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
)



const (
	streamName     = "ORDERS"
	streamSubjects = "ORDERS.*"
	subjectName    = "ORDERS.created"
)

const (
	subSubjectName ="ORDERS.created"
	pubSubjectName ="ORDERS.approved"
 
 )

type task struct {
    ID     string  `json:"id"`
    Parameters  string  `json:"parameters"`
    Status string  `json:"status"`
    Result  string `json:"result"`
}


func main() {

    nc, _ := nats.Connect(nats.DefaultURL)
    // nc, _ := nats.Connect("cakework-nats-cluster.internal")

	// Creates JetStreamContext
	js, err := nc.JetStream()
	checkErr(err)
   
	// Create Pull based consumer with maximum 128 inflight.
   // PullMaxWaiting defines the max inflight pull requests.


   for {
		fmt.Println("start of new for loop") 
		// Q: should we be creating a new pullsubscribe each time?
		sub, _ := js.PullSubscribe(subSubjectName, "order-review", nats.PullMaxWaiting(128))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	 
	    defer cancel()
      /*select {
      case <-ctx.Done():
		 fmt.Println("ctx is done")
		 fmt.Println("sleeping for 1 second")
		 time.Sleep(1 * time.Second)
        //  return
      default:
      }*/
      msgs, _ := sub.Fetch(10, nats.Context(ctx))
      for _, msg := range msgs {
         msg.Ack()
         var order Order
         err := json.Unmarshal(msg.Data, &order)
         if err != nil {
            log.Fatal(err)
         }
         log.Println("order-review service")
         log.Printf("OrderID:%d, CustomerID: %s, Status:%s\n", order.OrderID, order.CustomerID, order.Status)
         reviewOrder(js,order)
      }
	  fmt.Println("Done with for loop, starting new one")
   }



    gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
    // router.GET("/albums", getAlbums)
    // router.GET("/albums/:id", getAlbumByID)
    
    // router.GET("/get-status", getStatus)
    // router.GET("/get-result", getResult)
    router.Run()
}


type Order struct {
	OrderID    int
	CustomerID string
	Status     string
}



func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// reviewOrder reviews the order and publishes ORDERS.approved event
func reviewOrder(js nats.JetStreamContext, order Order) {
	// Changing the Order status
	order.Status ="approved"
	// orderJSON, _ := json.Marshal(order)

	// _, err := js.Publish(pubSubjectName, orderJSON)
	// if err != nil {
	//    log.Fatal(err)
	// }
	log.Printf("Order with OrderID:%d has been %s\n",order.OrderID, order.Status)
 }