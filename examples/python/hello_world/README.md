# Quick Start #
## Install the cli ##
`curl -L https://raw.githubusercontent.com/usecakework/cakeworkctl/main/install.sh | sh`

## Create a new project ##
``` 
cakework new my-app-name --lang python
cd my-app-name
```

## Optional: set up a virtual environment ##
```
python3 -m venv env
source env/bin/activate
```

## Install cakework module ##
`pip3 install cakework`

## Start your app ##
`cakework start`

## Calling the app ##
Refer to the tst/invoke_task.py for an example of how to invoke your running task.
```
python3 invoke_task.py
```

================
Below needs to be updated; ignore



# note: Example only works for Python at the moment

Download the cakeworkctl cli # TODO add instructions
`cd cakework-examples/examples/python/hello_world`

Modify `register_activity.py` so that you pass in your user id, app name, and activity name. # TODO: make it so that you don't need to pass in the user id, app name, and activity name via both the cli as well as via code.

Specify the user id, app name, and activity name, and the full path to the directory containing the register_activity.py code. In the future user id will be inferred once the user logs in; for now, pass it in manually.

Only lower case, numbers, and hyphens are allowed. 

Make sure that the activity name matches the name of the function that you're invoking as part of an activity.

Example:
`cakework deploy app activity /Users/jessieyoung/workspace/cakework-examples/examples/python/hello_world`

Modify `call_activity.py` to pass the parameters that you wish.

Note: if you want to deploy the sample app, you can just leave `register_activity.py` and `call_activity.py` as is.

Local testing:

Create a virtual env:
`python3 -m venv env` 
`source env/bin/activate`

```
pip install -r requirements.txt
python register_activity.py
cd build/activity
```

To test locally, start the server:
`python activity_server.py`

In a different terminal, start the virtual env and call the activity:
`python call_activity.py`

# TODO: need to add the instructions for using the cli to actually do the deployment to a remote host

