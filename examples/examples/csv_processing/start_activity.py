from sahale.client import Client
import json

async def main():
    url = "https://my-bucket.s3.amazonaws.com/file.csv"
    client = Client()
    id = client.start_new_activity("csv-processor", url)
    result = client.get_activity(id)
    if result.status == 'SUCCEEDED':
        logging.info(f"Activity succeeded with result: = '{json.dumps(result.output)}'.")

if __name__ == "__main__":
    main()
