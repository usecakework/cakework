from sahale.client.activity import Client
from sahale.app import App
import logging
import time
import csv
import requests
import json

def csv_processor(url):
    count = 0
    with requests.Session() as s:
        download = s.get(url)
        decoded_content = download.content.decode('utf-8')
        cr = csv.reader(decoded_content.splitlines(), delimiter=',')
        my_list = list(cr)
        for row in my_list:
            count += int(row[0])
    return count


async def main():
    # create and deploy the activity
    app = App()
    app.register_activity(csv_processor, "CsvProcessor")
    app.start()

    # invoking the activity
    client = Client()
    id = await client.start_new("CsvProcessor", request)
    logging.info(f"Started activity with ID = '{id}'.")

    # check the status of the activity invocation
    time.sleep(3)
    response = client.get_status(id)
    if response.status == 'SUCCEEDED':
         # output is the JSON-serialized out from csv_processor()
         # for example, "3" (since the output of csv_processor is an int)
        logging.info(f"Activity succeeded with result: = '{response.output}'.")

if __name__ == "__main__":
    main()