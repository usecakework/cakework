# Serverless Image Generation

This example shows how you can build a Python app that generates an image from text and uploads that image to S3. The image generation task runs serverlessly using Cakework and Banana. The example also shows how you can use Cakework in a webapp, that we built using Pynecone.

Check out the [Cakework docs](https://docs.cakework.com/) to learn more!

# Prerequisites
1. [Install Cakework CLI](https://docs.cakework.com/reference/cli/installation)
2. [Run a Stable Diffusion model on Banana](https://www.banana.dev/stable-diffusion)
3. Have an AWS account with a public S3 bucket

## To use the Web UI

1. [Install Pynecone](https://pynecone.io/docs/getting-started/installation)

# Setup
1. Update app/src/main.py with your AWS and Banana secrets.
2. Update client/src/main.py with your S3 bucket URL.

## To use the Web UI

1. Update frontend/src/main.py with your S3 bucket URL.

# Run the Cakework App

Sign up for Cakework and deploy the app from the ```app``` directory

```
cakework signup
pip3 install -r requirements.txt
cakework deploy
```

# Run the Cakework Client 

You can run the client from the ```client``` directory:

```
pip3 install -r requirements.txt
python3 src/main.py
```

# Run the Webapp

From the ```frontend``` directory:

```
pip3 install requirements.txt
pc run
```

Go to ```localhost:3000``` in your browser and start generating some images!


