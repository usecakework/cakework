from sahale.client.activity import Client
import json

async def main():
    client = Client()
    id = client.start_new("csv-processor", request)
    result = client.get_status(id)
    if result.status == 'SUCCEEDED':
        logging.info(f"Activity succeeded with result: = '{json.dumps(result.output)}'.")

if __name__ == "__main__":
    main()
