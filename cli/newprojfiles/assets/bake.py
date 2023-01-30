from cakework import Client
import time
from pathlib import Path

if __name__ == "__main__":
    p = Path(__file__).with_name('.env')
    with p.open('r') as f:
        CAKEWORK_CLIENT_TOKEN = f.readline().strip('\n')

        client = Client("REPLACE_APPNAME", CAKEWORK_CLIENT_TOKEN, local=False)

        run_id = client.run("say-hello", {"name":"from Cakework"}, compute={})
        print("Your run id is " + run_id)
        
        status = client.get_run_status(run_id)

        while status == "PENDING" or status == "IN_PROGRESS":
            print("Still baking...!")
            time.sleep(1)
            status = client.get_run_status(run_id)
        
        if status == "SUCCEEDED":
            result = client.get_run_result(run_id)
            print(result)
        else:
            print("Task stopped  with status: " + status)
