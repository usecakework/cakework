
generate the go server and clients:
```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
   proto/cakework/cakework.proto
```

sample postman request to start-task:
{
    "userId": "Shared",
    "app": "app",
    "task": "sAy_hello",
    "parameters": "[\"jessie\"]"
}
localhost:8080/start-task

go build -o frontend
export STAGE=dev && ./frontend
