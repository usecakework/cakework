Building the Docker image:
`docker build -t deploy:latest . --platform linux/amd64`

Running the service locally:
The service requires the Fly api token to be set. In prod, the token set using Fly Secrets. When testing the dockerized Go service locally, inject the token as an env variable, i.e.  
`docker run --env FLY_API_TOKEN=$REPLACE_ME -it --rm -p 8080:8080 deploy:latest`

If you already have the fly cli configured (logged in with the right credentials) set up on your local machine, you can run the service without docker (which may be slightly faster):
`go run app.go`