This example shows how you can build a python app generates an image from text using stable diffusion, and uploads that image to S3. It also shows an example of doing this with a webapp. The image generation task runs serverlessly using Cakework and Banana, and returns an ID that you can use to track the status of an app.

Check out the [Cakework docs](https://docs.cakework.com/) to learn more!

# Prerequisites
1. [Install Cakework CLI](https://docs.cakework.com/reference/cli/installation)
2. [Run a Stable Diffusion model on Banana](https://www.banana.dev/stable-diffusion)
3. Have an AWS account with a public S3 bucket

(To use the Web UI)
4. [Install Pynecone](https://pynecone.io/docs/getting-started/installation)

# Setup
1. Update app/src/main.py with your AWS and Banana secrets.
2. Update client/src/main.py with your S3 bucket URL.

(To use the Web UI)
3. Update frontend/src/main.py with your S3 bucket URL.

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


