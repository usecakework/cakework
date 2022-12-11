from sahale.client import Client
import json

async def main():
    url = "https://my-bucket.s3.amazonaws.com/file.csv"
    client = Client("my-app", "my-user-id") # TODO remove user id parameter and infer this from the logged-in credentials isntead 
    id = client.start_new_activity("csv-processor", url)
    result = client.get_activity(id)
    if result.status == 'SUCCEEDED':
        logging.info(f"Activity succeeded with result: = '{json.dumps(result.output)}'.")

if __name__ == "__main__":
    main()
