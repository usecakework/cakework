package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const LOGTAIL_QUERY_URL = "https://logtail.com//api/v1/query"

// TODO REMOVE ME
const LOGTAIL_TOKEN = "ou23pqL941JaKELaGiVbCARf"

const QUERY_QUERY_PARAM = "query"

// for now, just do a simplequery that gets everything and return the json response as string directly
// figure out the interface later
// figure out pagination later
func getLogs(simpleQuery string) (*RequestLogs, error) {
	req, err := http.NewRequest("GET", LOGTAIL_QUERY_URL, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Add(QUERY_QUERY_PARAM, "973879c4")
	req.URL.RawQuery = query.Encode()
	req.Header.Add("Authorization", "Bearer "+LOGTAIL_TOKEN)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var requestLogs RequestLogs
	json.Unmarshal(body, &requestLogs)

	fmt.Println(requestLogs)

	return &requestLogs, nil
}
