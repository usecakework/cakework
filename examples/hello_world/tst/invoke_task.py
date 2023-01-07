from cakework import Client
import time

CAKEWORK_CLIENT_TOKEN = "YOUR_CLIENT_TOKEN_HERE"

if __name__ == "__main__":
    client = Client("hello_world", CAKEWORK_CLIENT_TOKEN)
    
    request_id = client.say_hello("jessie")
    
    status = client.get_status(request_id)
    while (status == "PENDING" or status == "IN_PROGRESS"):
        print("in progress")
        status = client.get_status(request_id)
        time.sleep(1)

    if (client.get_status(request_id) == "SUCCEEDED"):
        result = client.get_result(request_id)
        print(result)