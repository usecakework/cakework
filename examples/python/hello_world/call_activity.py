from __future__ import print_function

import logging

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
