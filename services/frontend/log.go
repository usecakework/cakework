package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"

	"github.com/spf13/viper"
	"github.com/usecakework/cakework/lib/types"
)

const LOGTAIL_QUERY_URL = "https://logtail.com//api/v1/query"

const QUERY_QUERY_PARAM = "query"

// for now, just do a simplequery that gets everything and return the json response as string directly
// figure out the interface later
// figure out pagination later
func getLogs(simpleQuery string) (*types.RequestLogs, error) {
	LOGTAIL_TOKEN := viper.GetString("LOGTAIL_TOKEN")
	req, err := http.NewRequest("GET", LOGTAIL_QUERY_URL, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Add(QUERY_QUERY_PARAM, simpleQuery)
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

	var requestLogs types.RequestLogs
	json.Unmarshal(body, &requestLogs)

	// sort everything by timestamp
	sort.Slice(requestLogs.LogLines, func(i, j int) bool {
		return requestLogs.LogLines[i].Timestamp < requestLogs.LogLines[j].Timestamp
	})

	return &requestLogs, nil
}
