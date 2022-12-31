
generate the go server and clients:
```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
   proto/cakework/cakework.proto
```

go run main.go -local

TODO: figure out why we need to manually modify the generated cakework files