package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/auth"
)

// takes as input a struct, adds auth headers
func CallV2(url string, method string, reqStruct interface{}, provider auth.CredentialsProvider) (*http.Response, error) {
	jsonReq, err := json.Marshal(reqStruct)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonReq))

	if err != nil {
		return nil, err
	}

	creds, err := provider.GetCredentials()
	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, err
	}

	if creds.Type == "BEARER" {
		req.Header.Add("Authorization", "Bearer "+creds.AccessToken)
		req.Header.Add("Content-Type", "application/json")
	} else if creds.Type == "API_KEY" {
		req.Header.Add("X-Api-Key", creds.ApiKey)
		req.Header.Add("Content-Type", "application/json")
	} else {
		log.Debug("Credentials type is neither bearer nor api key. Not adding auth headers")
	}

	return CallHttpV2(req)
}

// takes *http.Request, does perform auth
func CallHttpAuthedV2(req *http.Request, provider auth.CredentialsProvider) (*http.Response, error) {
	creds, err := provider.GetCredentials()
	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, err
	}

	if creds.Type == "BEARER" {
		log.Debug("Adding bearer token to header")
		req.Header.Add("Authorization", "Bearer "+creds.AccessToken)
		req.Header.Add("Content-Type", "application/json")
	} else if creds.Type == "API_KEY" {
		log.Debug("Adding API key to header")
		req.Header.Add("X-Api-Key", creds.ApiKey)
		req.Header.Add("Content-Type", "application/json")
	} else {
		log.Debug("Credential type is neither bearer nor api key; request will not be authed")
	}

	return CallHttpV2(req)

}

// takes *http.Request, does not perform auth
// not really ideal, remember to close when you use this
func CallHttpV2(req *http.Request) (*http.Response, error) {
	// fmt.Println(PrettyPrintRequest(req)) // TODO delete
	client := http.Client{
		Timeout: time.Second * 60,
	}

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	// fmt.Println(PrettyPrintResponse(res)) // TODO delete
	return res, nil
}
