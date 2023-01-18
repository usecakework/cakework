# # syntax=docker/dockerfile:1

# FROM golang:1.19
# WORKDIR /app
# COPY . .
# RUN go mod download
# RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o frontend .

# syntax=docker/dockerfile:1
FROM golang:1.19-alpine
ENV GO111MODULE=on
WORKDIR /app

# RUN go mod download

COPY . . 
RUN go build -o poller
# RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o poller .

FROM alpine:latest  
WORKDIR /root/
COPY --from=0 /app/poller ./
EXPOSE 8080
CMD ["./poller"]