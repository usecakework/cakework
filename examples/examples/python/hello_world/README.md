Create a virtual env:
`python3 -m venv env` 
`source env/bin/activate`

```
pip install -r requirements.txt
python register_activity.py
cd build/activity
```

To test locally, start the server:
`python activity_server.py`

In a different terminal, start the virtual env and call the activity:
`python call_activity.py`


