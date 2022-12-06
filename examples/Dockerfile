# syntax=docker/dockerfile:1

FROM golang:1.19-alpine as builder

ENV GO111MODULE=on
WORKDIR /app
ENV GIN_MODE=release
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o app app.go
# EXPOSE 8080
# RUN apt-get update; apt-get install curl
# RUN curl -L https://fly.io/install.sh | sh
# CMD ["ls", "/usr/bin"]
# CMD ["./app"]
# TODO use a slimmer alpine image to run the service?

FROM alpine:latest as runner
WORKDIR /root/
RUN apk update && apk add curl
RUN curl -L https://fly.io/install.sh | sh
ENV FLYCTL_INSTALL=/root/.fly
ENV PATH=$FLYCTL_INSTALL/bin:$PATH

# TODO need to remove this!!! 
ENV FLY_API_TOKEN=5hLSL0gwiWBbhjiGC0u0mncbKlwT--OOutbf4DCO01Y 
COPY --from=builder /app/app .
# RUN fly machines api-proxy --org sahale &
# RUN curl -i -X POST -H "Authorization: Bearer ${FLY_API_TOKEN}" -H "Content-Type: application/json" http://${FLY_API_HOSTNAME}/v1/apps/jessie-activity-test/machines/e148e471be0789/start
EXPOSE 8080
CMD ["./app"]