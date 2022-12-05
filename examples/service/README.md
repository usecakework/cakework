Building the Docker image:
`docker build -t service:latest . --platform linux/amd64`

Running the service:
`docker run -it --rm -p 8080:8080 service`