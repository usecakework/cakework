# Serverless Image Generation

This example shows how you can build a Python app that generates an image from text and uploads that image to S3. The image generation task runs serverlessly using Cakework and [Banana](https://banana.dev). The example also shows how you can use Cakework in a web UI that we built using [Pynecone](https://pynecone.io/).

Check out the [Cakework docs](https://docs.cakework.com/) to learn more!

# Prerequisites
1. [Install Cakework CLI](https://docs.cakework.com/reference/cli/installation).
2. Sign up for Cakework with ```cakework signup```.
3. [Run a Stable Diffusion model on Banana](https://www.banana.dev/stable-diffusion).
4. Have an AWS account with a public S3 bucket.

# Deploy the App
1. Update ```app/src/main.py``` with your AWS and Banana secrets.
2. Deploy the app from the ```app``` directory with ```cakework deploy```.

# Run tasks from the Python script
1. Generate a Cakework client token with ```cakework create-client-token my-token```.
2. Update ```client/src/main.py``` with your newly generated Cakework client token.
3. Update ```client/src/main.py``` with your S3 bucket URL.
4. Run the script from the ```client``` directory:

```
pip3 install -r requirements.txt
python3 src/main.py
```

# Run tasks from the Web UI
1. [Install Pynecone](https://pynecone.io/docs/getting-started/installation)
2. Generate a Cakework client token with ```cakework create-client-token my-token```.
3. Update ```client/src/main.py``` with your Cakework client token.
4. Update ```frontend/src/main.py``` with your S3 bucket URL.
5. Install dependencies and start the webapp.

```
pip3 install -r requirements.txt
pc run
```

6. Go to ```localhost:3000``` in your browser and start making some pictures!

# Modifying the App
Have fun and check out the [docs](https://docs.cakework.com/). Make sure you create a virtual env for the App before you deploy to Cakework.