package main

// Q: should we just expose a gin gonic server with no methods?
import (
	pb "cakework/poller/proto/cakework"
	"context"
	"encoding/json"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

// const (
// 	streamName     = "ORDERS"
// 	streamSubjects = "ORDERS.*"
// 	subjectName    = "ORDERS.created"
// )

const (
	subSubjectName ="TASKS.created"
	// pubSubjectName ="ORDERS.approved"
 
 )

 type TaskRun struct {
    UserId     string  `json:"userId"`
	App	string `json:"app"`
	Task string `json:"task"`
    Parameters  string  `json:"parameters"`
	RequestId     string  `json:"requestId"`
	Status	string `json:"status"`
	Result  string  `json:"result"`
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
		// fmt.Println("starting new pull subscribe") 
		// Q: should we be creating a new pullsubscribe each time?
		sub, _ := js.PullSubscribe(subSubjectName, "task-review", nats.PullMaxWaiting(128))
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
         var taskRun TaskRun
         err := json.Unmarshal(msg.Data, &taskRun)
         if err != nil {
            log.Fatal(err)
         }
         log.Printf("UserId:%s, App: %s, Task:%s, Parameters: %s, RequestId: %s, Status: %s, Result: %s\n", taskRun.UserId, taskRun.App, taskRun.Task, taskRun.Parameters, taskRun.RequestId, taskRun.Status, taskRun.Result)
         runTask(js, taskRun)
      }
   }
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// reviewOrder reviews the order and publishes ORDERS.approved event
func runTask(js nats.JetStreamContext, taskRun TaskRun) {
	var conn *grpc.ClientConn
    // conn, err := grpc.Dial("shared-app-say-hello.fly.dev:443", grpc.WithInsecure())
	// endpoint := "localhost:50051"
	endpoint := taskRun.UserId + "-" + taskRun.App + "-" + taskRun.Task + ".fly.dev:443" // TODO replace this with the actual name of the fly task (uuid)

    conn, err := grpc.Dial(endpoint, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := pb.NewCakeworkClient(conn)

	createReq := pb.Request{ Parameters: taskRun.Parameters }
	response, err := c.RunActivity(context.Background(), &createReq)
	if err != nil {
		log.Fatalf("Error Cakework RunActivity: %s", err)
	} else {
		// successfully submitted; move to IN_PROGRESS
		// note: the fly python grpc worker probably still need to be able to update the status
	}
	log.Printf("Response from worker: %s", response.Result)

	
	// Changing the Order status
	// order.Status ="approved"

	// TODO call fly
	// for now, have this be synchronous and wait for fly results. in the future, do a fire and forget
	// orderJSON, _ := json.Marshal(order)

	// _, err := js.Publish(pubSubjectName, orderJSON)
	// if err != nil {
	//    log.Fatal(err)
	// }
	// log.Printf("Order with OrderID:%d has been %s\n",order.OrderID, order.Status)
	// TODO here we will call the fly endpoint
 }