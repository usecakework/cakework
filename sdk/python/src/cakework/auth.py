import http.client
import json
from cakework.exceptions import CakeworkError


# TODO remove the client secret from here for open source version
# takes scope as a string with APIs separated by a space
def get_token(client_id, client_secret, scope):
    conn = http.client.HTTPSConnection("dev-qanxtedlpguucmz5.us.auth0.com")

    payload = "{\"client_id\":\"" + client_id + "\",\"client_secret\":\"" + client_secret + "\",\"audience\":\"https://cakework-frontend.fly.dev\",\"grant_type\":\"client_credentials\",\"scope\":\"" + scope + "\"}"
    headers = { 'content-type': "application/json" }

    conn.request("POST", "/oauth/token", payload, headers)

    res = conn.getresponse()
    data = res.read()

    decoded = json.loads(data.decode("utf-8"))
    if "access_token" in decoded:
        return decoded["access_token"]
    else:
        if "error" in decoded:
            print(decoded["error"])
        if "error_description" in decoded:
            print(decoded["error_description"])
        raise CakeworkError("Failed to fetch access token")

# TODO fetch refresh token