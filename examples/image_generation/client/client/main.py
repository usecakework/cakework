from cakework import Client
import time
import json

S3_BUCKET_URL = "YOUR_S3_BUCKET_URL"
CAKEWORK_CLIENT_TOKEN = "YOUR_CAKEWORK_CLIENT_TOKEN"

if __name__ == "__main__":
    client = Client("image_generation", CAKEWORK_CLIENT_TOKEN)
    
    # You can persist this request ID to get status of the job later
    request_id = client.generate_image("cute piece of cake", "cartoon")
    print(request_id)

    status = client.get_status(request_id)
    while (status == "PENDING" or status == "IN_PROGRESS"):
        print("Still baking that cake!")
        status = client.get_status(request_id)
        time.sleep(1)

    if (client.get_status(request_id) == "SUCCEEDED"):
        result = json.loads(client.get_result(request_id))
        url = S3_BUCKET_URL + result["s3Location"]
        print(url)
