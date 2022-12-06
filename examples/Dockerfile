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
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]