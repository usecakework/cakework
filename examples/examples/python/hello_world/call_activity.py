# TODO delete this. this is only for testing a particular fly endpoint

# Note: this is for testing only. 

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
"""The Python implementation of the GRPC helloworld.Greeter client."""

from __future__ import print_function

import logging

# from cakework.client import Client #TODO change bak to this
from cakework.client import Client
import json

def run():
    client = Client("app")
    parameters = {"name": "jessie", "age": 34}
    print("Starting activity with parameters: " + json.dumps(parameters))
    response = client.start_new_activity("activity", parameters) # for now, synchronously wait for result
    print("Got result: " + response.result)

if __name__ == '__main__':
    logging.basicConfig()
    run()
