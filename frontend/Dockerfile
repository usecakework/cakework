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

FROM alpine:latest as runner
WORKDIR /root/
RUN apk update && apk add curl
RUN curl -L https://fly.io/install.sh | sh
ENV FLYCTL_INSTALL=/root/.fly
ENV PATH=$FLYCTL_INSTALL/bin:$PATH
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]