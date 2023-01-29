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
        # TODO get rid of "shared"
        self.project = project.lower().replace('_', '-')
        # make a call to the frontend to get the user_id using the client token
        if local:
            self.frontend_url = "http://localhost:8080"
        else:
            self.frontend_url = "https://cakework-frontend.fly.dev"

        response = None
        self.headers = None
        try:

            # based on the particular user id, how do we fetch user credentials and get a token on behalf of the user?
            # need SOME sort of token first
            self.headers = {'content-type': 'application/json', 'X-Api-Key': client_token, 'project': project}
            response = requests.get(f"{self.frontend_url}/client/user-from-client-token", json={"token": client_token}, headers=self.headers)      # Q: should this take app?        
               
            # if dont' get a a valid user
            # TODO raise exception
                       
            # TODO: handle error for if can't get a user from the client token. technically
            # we should be getting a JWT token instead first
            response.raise_for_status()
            # TODO: handle http error, or request id not found error
        except requests.exceptions.HTTPError as err:
            if err.response.status_code == 401:
                raise exceptions.CakeworkError("Missing client token; please generate and add a client token to your client.") from err
            else:
                raise exceptions.CakeworkError("Http error while connecting to Cakework frontend") from err
        except requests.exceptions.Timeout as err:
            raise exceptions.CakeworkError("Timed out connecting to Cakework frontend") from err
        except requests.exceptions.RequestException as err:
            raise exceptions.CakeworkError("Request exception connecting to Cakework frontend") from err
        except (ConnectionRefusedError, ConnectionResetError) as err:
            raise exceptions.CakeworkError("Failed to connect to Cakework frontend service") from err
        except Exception as err:
            # TODO catch and raise specific errors? 
            raise exceptions.CakeworkError("Error happened while initializing Cakework client") from err
        if response is not None:
            if response.status_code == 200:
                response_json = response.json()
                self.user_id = response_json["id"] # Q: how to return None if process is still executing? instead of empty string
            else:
                # TODO add logging to figure out what happened
                raise exceptions.CakeworkError("Error happened while trying to get the user from the client token")
        else:
            raise exceptions.CakeworkError("Error happened while trying to get the user from the client token")

        self.local = local
        
    def get_run_status(self, run_id):
        response = None
        try:
            # Q: status 200 vs 201??? what's the diff?
            # TODO strip app from everywhere
            response = requests.get(f"{self.frontend_url}/client/runs/{run_id}/status", json={"userId": self.user_id, "project": self.project, "runId": run_id}, headers=self.headers)                
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
                return status
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
            response = requests.get(f"{self.frontend_url}/client/runs/{run_id}/result", json={"userId": self.user_id, "project": self.project, "runId": run_id}, headers=self.headers)                
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
                result = response.text
                return result
            elif response.status_code == 404:
                return None
            else:
                raise exceptions.CakeworkError("Internal server exception")
        else:
            raise exceptions.CakeworkError("Internal server exception") 
    
    def run(self, project, task, params, compute):
        sanitized_task = task.lower()
        sanitized_task = task.replace('_', '-')

        request = {
            "parameters": params
        }

        cpu = compute.get("cpu")
        if cpu is not None:
            if cpu < 1 or cpu > 8:
                raise exceptions.CakeworkError("Number of cpus must be between 1 and 8")
            else:
                request["cpu"] = cpu
                
        memory = compute.get("memory")
        if memory is not None:
            if memory < 256 or memory > 16384:
                raise exceptions.CakeworkError("Amount of memory must be between 256 and 16384 mb")
            else:
                request["memory"] = memory
    
        # print(request) # TODO delete

        response = requests.post(f"{self.frontend_url}/client/projects/{project}/tasks/{sanitized_task}/runs", json=request, headers=self.headers)
        response_json = response.json()
        run_id = response_json["runId"] # this may be null?
        return run_id

    def __getattr__(self, name):
        def method(*args, compute={}):            
            sanitized_name = name.lower()
            sanitized_name = name.replace('_', '-')

            parameters = json.dumps(args)
            request = {
                "parameters": parameters
            }

            cpu = compute.get("cpu")
            if cpu is not None:
                if cpu < 1 or cpu > 8:
                    raise exceptions.CakeworkError("Number of cpus must be between 1 and 8")
                else:
                    request["cpu"] = cpu
                    
            memory = compute.get("memory")
            if memory is not None:
                if memory < 256 or memory > 16384:
                    raise exceptions.CakeworkError("Amount of memory must be between 256 and 16384 mb")
                else:
                    request["memory"] = memory
        
            # print(request) # TODO delete
  
            response = requests.post(f"{self.frontend_url}/client/projects/{self.project}/tasks/{sanitized_name}/runs", json=request, headers=self.headers)
            response_json = response.json()
            run_id = response_json["runId"] # this may be null?
            return run_id

        return method