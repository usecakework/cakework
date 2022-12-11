from sahale.app import App
import logging
import csv
import requests

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

app = App("my-cool-app")
app.register_activity(csv_processor, "csv-processor")
