from cakework import Client
import time
from pathlib import Path

if __name__ == "__main__":
    p = Path(__file__).with_name('.env')
    with p.open('r') as f:
        CAKEWORK_CLIENT_TOKEN = f.readline().strip('\n')

        client = Client("` + appName + `", CAKEWORK_CLIENT_TOKEN)

        # You can persist this request ID to get status of the job later
        request_id = client.say_hello("from Cakework")
        print("Your request id is " + request_id)

        status = client.get_status(request_id)
        while (status == "PENDING" or status == "IN_PROGRESS"):
            print("Still baking...!")
            status = client.get_status(request_id)
            time.sleep(1)

        if (client.get_status(request_id) == "SUCCEEDED"):
            result = client.get_result(request_id)
            print(result)