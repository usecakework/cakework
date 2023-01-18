# # syntax=docker/dockerfile:1

# FROM golang:1.19
# WORKDIR /app
# COPY . .
# RUN go mod download
# RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o frontend .

# syntax=docker/dockerfile:1
FROM golang:1.19-alpine
WORKDIR /app

# RUN go mod download

COPY . . 
RUN go build -o /frontend
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o frontend .

FROM alpine:latest  
WORKDIR /root/
COPY --from=0 /app/frontend ./
EXPOSE 8080
CMD ["./frontend"]