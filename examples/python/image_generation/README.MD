This example shows how you can build a webapp that generates an image from text using stable diffusion, and uploads that image to S3. The image generation task happens asynchronously and will persist through the user leaving the page. The entire task runs serverlessly through Cakework and Banana.

Check out the [Cakework docs](https://docs.cakework.com/) to learn more!

# Prerequisites
1. [Install Cakework](https://docs.cakework.com/reference/cli/installation)
2. [Install Pynecone](https://pynecone.io/docs/getting-started/installation)
3. [Run a Stable Diffusion Model on Banana](https://www.banana.dev/stable-diffusion)
4. Have an AWS account
5. Update app/main.py with your AWS and Banana secrets.

# Run the Cakework App

From the ```app``` directory:

```
pip3 install requirements.txt
cakework login
cakework deploy
```

You can test the task and see basic Cakework functionality (run a task, get status, get result) by running the test client from the ```app``` directory:

```
python3 src/test_client.py
```

# Run the Webapp

From the ```frontend``` directory:

```
pip3 install requirements.txt
pc run
```

Go to ```localhost:3000``` in your browser and start generating some images!


