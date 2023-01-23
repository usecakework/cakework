# Cakework

Cakework is a purpose-built cloud for serverless async backends. Cakework is great for operations that take time or more compute such as file processing or machine learning. 

# Self hosting

1. Set up a MySQL DB with the following [schema](db/schema.prisma).

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
3. Run the frontend. Create a .env file with your secrets. From the ```/frontend``` dir:
```
go build -o frontend
export STAGE=dev && ./frontend
```
You should now be able to hit the frontend and start tasks, etc.

4. Run the poller.
Wireguard into your Fly account by following the instructions [here](https://fly.io/docs/reference/private-networking/).

```
go build -o poller
export STAGE=dev && ./poller
```