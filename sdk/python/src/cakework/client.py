from __future__ import print_function

# import logging

import json
import sys
import uuid
from random import randrange # TODO remove this
import requests
import logging
from cakework import exceptions
from urllib3.exceptions import NewConnectionError
import os

# TODO: need to re-enable TLS for the handlers in the fly.toml file. Try these settings: https://community.fly.io/t/urgent-grpc-server-unreachable-via-grpcurl/2694/12 for alpn
# TODO figure out how to configure the settings for fly.toml for grpc!
# TODO also need to make sure different runs don't interfere with each other
# TODO add a parameter for an entry point into the system (currently, assume that using cakework_app.py)
logging.basicConfig(level=logging.INFO)

class Client:
    def __init__(self, project, client_token, local=False): # TODO: infer user id // TODO revert local back to False
        self.project = project
        self.client_token = client_token

        if local:
            self.frontend_url = "http://localhost:8080"
        else:
            self.frontend_url = "https://cakework-frontend.fly.dev"

        self.local = local
        
    def get_run_status(self, run_id):
        response = None
        try:
            # Q: status 200 vs 201??? what's the diff?
            # TODO strip app from everywhere
            response = requests.get(f"{self.frontend_url}/client/runs/{run_id}/status", params={"token": self.client_token})                
            response.raise_for_status()
            # TODO: handle http error, or request id not found error
        except requests.exceptions.HTTPError as err:
            raise exceptions.CakeworkError("Http error while connecting to Cakework frontend") from err
        except requests.exceptions.Timeout as err:
            raise exceptions.CakeworkError("Timed out connecting to Cakework frontend") from err
        except requests.exceptions.RequestException as err:
            raise exceptions.CakeworkError("Request exception connecting Cakework frontend") from err
        except (ConnectionRefusedError, ConnectionResetError) as err:
            raise exceptions.CakeworkError("Failed to connect to Cakework frontend service") from err
        except Exception as err:
            # TODO catch and raise specific errors? 
            raise exceptions.CakeworkError("Error happened while getting status") from err
        if response is not None:
            if response.status_code == 200:
                status = response.text
                return json.loads(status)
            elif response.status_code == 404:
                return None
            else:
                raise exceptions.CakeworkError("Internal server exception")
        else:
            raise exceptions.CakeworkError("Internal server exception") 

    # TODO figure out how to refactor get_result and get_status  
    def get_run_result(self, run_id):
        response = None
        try:
            # Q: status 200 vs 201??? what's the diff?
            response = requests.get(f"{self.frontend_url}/client/runs/{run_id}/result", params={"token": self.client_token})                
            response.raise_for_status() # TODO delete this?
            # TODO: handle http error, or request id not found error
        except requests.exceptions.HTTPError as errh:
            raise exceptions.CakeworkError("Http error while connecting to Cakework frontend")
        except requests.exceptions.Timeout as errt:
            raise exceptions.CakeworkError("Timed out connecting to Cakework frontend")
        except requests.exceptions.RequestException as err:
            raise exceptions.CakeworkError("Request exception connecting Cakework frontend")
        except (ConnectionRefusedError, ConnectionResetError) as e:
            raise exceptions.CakeworkError("Failed to connect to Cakework frontend service")
        except Exception as e:
            # TODO catch and raise specific errors? 
            raise exceptions.CakeworkError("Something unexpected happened")
        if response is not None:
            if response.status_code == 200:
                result = json.loads(response.json())
                return result
            elif response.status_code == 404:
                return None
            else:
                raise exceptions.CakeworkError("Internal server exception")
        else:
            raise exceptions.CakeworkError("Internal server exception") 
    
    def run(self, task, params, compute ={"cpu":1, "memory": 256}):
        request = {
            "parameters": params,
            "compute": {}
        }

        cpu = compute.get("cpu")
        if cpu is not None:
            if cpu < 1 or cpu > 8:
                raise exceptions.CakeworkError("Number of cpus must be between 1 and 8")
            else:
                request["compute"]["cpu"] = cpu
        else:
            request["compute"]['cpu'] = 1
                
        memory = compute.get("memory")
        if memory is not None:
            if memory < 256 or memory > 16384:
                raise exceptions.CakeworkError("Amount of memory must be between 256 and 16384 mb")
            else:
                request["compute"]["memory"] = memory
        else:
            request["compute"]['memory'] = 256
    
        request["token"] = self.client_token
        response = requests.post(f"{self.frontend_url}/client/projects/{self.project}/tasks/{task}/runs", json=request, params={"token": self.client_token})
        response_json = response.json()
        if response.status_code == 201:
            run_id = response_json["runId"]
            return run_id
        elif response.status_code == 404:
            raise exceptions.CakeworkError("Task " + task + " for project " + self.project + " not found. Have you tried running `cakework deploy` first?")  
        else:
            print(response) # TODO delete? 
            raise exceptions.CakeworkError("Internal server exception")  