package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
)


func main() {
    // nc, _ := nats.Connect(nats.DefaultURL)
    nc, _ := nats.Connect("https://cakework-nats-cluster.fly.dev.internal")

    // Simple Publisher
    nc.Publish("foo", []byte("Hello World"))

    // Simple Async Subscriber
    nc.Subscribe("foo", func(m *nats.Msg) {
        fmt.Printf("Received a message: %s\n", string(m.Data))
    })

    nc.Publish("foo", []byte("Hello World 2"))

}

