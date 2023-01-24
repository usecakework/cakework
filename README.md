# Cakework

Cakework is a purpose-built cloud for serverless async backends. It shines at operations that take time or more compute such as file processing or machine learning. 

# Self hosting

1. Set up a MySQL DB with the following [schema](db/schema.prisma).
2. Sign up for a Fly.io account.
3. Start the log shipper (Requires Logtail account). From the `./log-shipper` dir:
    1. Modify the ```fly.toml``` file with your Fly.io org name
    2. [Set the appropriate secrets for Fly and Logtail](https://github.com/superfly/fly-log-shipper)
    3. Run ```fly deploy```
4. Set up Auth0. Configure your frontend service as an API with the appropriate scopes, the poller as an Application, and CLI as a Native Application.

# Set up local development

1. [Install NATS](https://docs.nats.io/nats-concepts/what-is-nats/walkthrough_setup).
2. Start NATS server with Jetstream.
```
nats-server -js
```
3. Run the frontend. Create a .env file with your secrets. From the `./frontend` dir, run:
```
make local
```
You should now be able to hit the frontend and start tasks, etc.

4. Run the poller.
Wireguard into your Fly account by following the instructions [here](https://fly.io/docs/reference/private-networking/).
From the `./poller` dir, run:

```
make local
```

# Deploy to Fly
1. Deploy a NATS cluster to Fly.io by using this project: https://github.com/fly-apps/nats-cluster. Note the app name that you select for your Fly App; you'll need this to configure the frontend and poller services.
2. Deploy frontend service
```
cd services/frontend
make deploy
```
Store the secrets in your .env file in Fly. You'll need to store an additional secret STAGE which should not be equal to "dev"
3. Deploy poller service
```
cd services/frontend
make deploy
```
Store the secrets in your .env file in Fly. You'll need to store an additional secret STAGE which should not be equal to "dev"

# Build the CLI
```
cd cli
go build -o cli
```
This create an executable called `cli`. You can create an alias in your .rc script so that invocations to `cakework` point to the path of the executable.
```
alias cakework="~/workspace/cakework/cli/cli"
```

# Help
We love questions and feedback! Come chat with us on [Discord](https://discord.gg/yB6GvheDcP) <3 or email us at eric at cakework dot com or jessie at cakework dot com