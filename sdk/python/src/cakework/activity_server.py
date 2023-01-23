# Copyright 2015 gRPC authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""The Python implementation of the GRPC helloworld.Greeter server."""

from concurrent import futures
import logging
import grpc
from cakework import cakework_pb2
from cakework import cakework_pb2_grpc
import json
import os
import threading
import requests
import logging
from dotenv import load_dotenv

logging.basicConfig(level=logging.INFO)

def get_token(context):
    metadict = dict(context.invocation_metadata())
    authorization = metadict['authorization']
    split = authorization.split(' ')
    return split[1]

class Cakework(cakework_pb2_grpc.CakeworkServicer):
    def __init__(self, user_activity, local=False):
        load_dotenv()
        self.user_activity = user_activity
        self.local = local
        if self.local:
            self.frontend_url = "http://localhost:8080"
        else:
            self.frontend_url = os.getenv("FRONTEND_URL")

    def RunActivity(self, request, context):
        token = get_token(context)
        scope = "update:status update:result"
        headers = {'content-type': 'application/json', 'authorization': 'Bearer ' + token}
        response = requests.patch(f"{self.frontend_url}/update-status", json={"userId": request.userId, "app": request.app, "requestId": request.requestId, "status": "IN_PROGRESS"}, headers=headers)
        # TODO check the response
        parameters = json.loads(request.parameters)
        # parameters = json.loads(request.parameters) # instead of passing a dict (parameters), pass a variable number of parameters 
        task = threading.Thread(target=self.background, args=[request.userId, request.app, request.requestId, parameters, token])
        task.daemon = True # Q: does returning kill the task?
        task.start()
        # what should we return? now, the client is no longer hitting the grpc server
        return cakework_pb2.Reply(result=json.dumps("worker started task")) # TODO this should return the request id

    # note: we need to get the request id in here as well
    def background(self, user_id, app, request_id, parameters, token):
        res = ""
        # logging.info("starting background task with parameters: " + str(parameters))
        try:
            logging.info("Starting request " + request_id)
            res = self.user_activity(*parameters)
            # TODO call the frontend api to update the status and result
            
            logging.info("Finished request " + request_id)
            headers = {'content-type': 'application/json', 'authorization': 'Bearer ' + token}
            response = requests.patch(f"{self.frontend_url}/update-result", json={"userId": user_id, "app": app, "requestId": request_id, "result": json.dumps(res)}, headers=headers)
            # TODO check the response
            response = requests.patch(f"{self.frontend_url}/update-status", json={"userId": user_id, "app": app, "requestId": request_id, "status": "SUCCEEDED"}, headers=headers)

        except Exception as e:
            logging.exception("Error occurred while running task", exc_info=True)
            # logging.info(e)
            response = requests.patch(f"{self.frontend_url}/update-status", json={"userId": user_id, "app": app, "requestId": request_id, "status": "FAILED"})
            # TODO: handle case if updating the status in the db failed; then user may keep polling forever
            return cakework_pb2.Reply(result=None)
        finally:
            logging.info("Request complete, exiting vm")
            os._exit(1)

        # Q: who are we returning to here? instead of returning, we can just write this to the database. or emit a message and have the poller be in charge of writing to db
        return cakework_pb2.Reply(result=json.dumps(res))

def serve():
    port = '50051'
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1)) # what should the default be?
    cakework_pb2_grpc.add_CakeworkServicer_to_server(Cakework(), server)
    server.add_insecure_port('[::]:' + port)
    server.start()
    logging.info("Server started, listening on " + port)
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig()
    serve()

class ActivityServer:
    def __init__(self, user_activity, local=False):
        self.user_activity = user_activity
        self.server = grpc.server(futures.ThreadPoolExecutor(max_workers=1)) # what should the default be?
        self.local = local

    def start(self):        
        port = '50051'
        cakework_pb2_grpc.add_CakeworkServicer_to_server(Cakework(self.user_activity, self.local), self.server)
        self.server.add_insecure_port('[::]:' + port)
        self.server.start()
        logging.info("Server started, listening on " + port)
        self.server.wait_for_termination()