
generate the go server and clients:
```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
   proto/cakework/cakework.proto
```

Running locally:
First spin up a nats jetstream endpoint on port 4222
Then run:
go build -o main && ./main -local

TODO: figure out why we need to manually modify the generated cakework files