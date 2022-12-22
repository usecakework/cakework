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
`python3 invoke_task.py`
