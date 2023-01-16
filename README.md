

# Set up your own stack


5. Start the log shipper (Requires Logtail account). From the ```./log-shipper`` dir:
    1. Modify the ```fly.toml``` file with your Fly.io org name
    2. [Set the appropriate secrets for Fly and Logtail](https://github.com/superfly/fly-log-shipper)
    3. Run ```fly deploy```

# Set up local development

1. [Install NATS](https://docs.nats.io/nats-concepts/what-is-nats/walkthrough_setup).
2. Start NATS server with Jetstream.
```
nats-server -js
```
3. Run the frontend. From the ```/frontend``` dir:
```
go build -o frontend
./frontend -local
```
4. Run the poller. From the ```./poller``` dir:
```
go build -o poller
./poller -local
```

You should be able to hit the frontend and start tasks, etc.