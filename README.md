# Cakework

<div align="center">
<img src="https://cakework-logo.s3.us-west-2.amazonaws.com/favicon.png" width="150">
</div>

Cakework helps you build serverless async backends with no cloud resources to manage. You launch a new backend in minutes, iterate with just your code and dependencies, and get everything you need to do devops. Each request to your backend runs on its own micro VM with no timeout limitations and CPU and memory you specify. Cakework is built for work that takes time or more compute such as file processing, data analysis, or report generation.

# Documentation

Check out the docs [here](https://docs.cakework.com/) to get started with Cakework.

# Community

Join the [Discord](https://discord.gg/yB6GvheDcP) or send us an [email](mailto:hi@cakework.com)!

# Why Cakework

## 🍰 Zero Infrastructure

Your backend is just code. We take care of queues, workers, and data behind the scenes.

## 🍰 Compute, Your Way

Set CPU and memory per request. Each request runs on its own microVM with no timeout limitations.

## 🍰 Client SDKs

Use the pre-built Client SDKs to run tasks, get status, and get results. No additional backend work required.

## 🍰 Built-in Devops.

Use the CLI to query requests by status, and view inputs, outputs, and logs.

# Get it Running!

## Account Signups
1. Sign up for a Fly.io account
2. Sign up for a Logtail account 
3. Sign up for an Auth0 account

## Setup 
1. Set up a hosted MySQL DB with the following [schema](db/schema.prisma). We use Planetscale.
2. Set up Auth0. Configure the frontend service as an API with the appropriate scopes, the poller as an Application, and CLI as a Native Application.

## Deploy
1. Deploy a NATS cluster to Fly.io by using this project: https://github.com/fly-apps/nats-cluster. Note the app name that you select for your Fly App; you'll need this to configure the frontend and poller services.

2. Deploy the frontend service
```
cd services/frontend
make deploy
```
Store all your secrets in Fly with all the appropriate variables. You'll need to store an additional secret STAGE which should not be equal to "dev".

3. Deploy the poller service
```
cd services/poller
make deploy
```
Store the secrets in your .env file in Fly. You'll need to store an additional secret STAGE which should not be equal to "dev"

4. Deploy the log shipper. From the `services/log-shipper` dir:
    1. Modify the ```fly.toml``` file with your Fly.io org name
    2. [Set the appropriate secrets for Fly and Logtail](https://github.com/superfly/fly-log-shipper)
    3. Run ```fly deploy```

## Build the CLI
```
cd cli
go build -o cli
```
This create an executable called `cli`. You can create an alias in your .rc script so that invocations to `cakework` point to the path of the executable.
```
alias cakework="~/workspace/cakework/cli/cli"
```

# Local Development

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

# Help
We love questions and feedback! Come chat with us on [Discord](https://discord.gg/yB6GvheDcP) <3 or email us at eric at cakework dot com or jessie at cakework dot com