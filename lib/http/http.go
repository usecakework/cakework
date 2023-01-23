package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"net/http/httputil"

	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/auth"
)

// TODO deprecate and use v2

// takes as input a struct, adds auth headers
func Call(url string, method string, reqStruct interface{}, provider auth.CredentialsProvider) (map[string]interface{}, *http.Response, error) {
	jsonReq, err := json.Marshal(reqStruct)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonReq))
	// reqDump, _ := httputil.DumpRequestOut(req, true) // can't always dump out the request if it's null? 

	// fmt.Println(string(reqDump)) //TODO delete

	if err != nil {
		return nil, nil, err
	}

	creds, err := provider.GetCredentials()

	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, nil, err
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

	return CallHttp(req)
}

// takes *http.Request, does perform auth
func CallHttpAuthed(req *http.Request, provider auth.CredentialsProvider) (bodyMap map[string]interface{}, res *http.Response, err error) {
	creds, err := provider.GetCredentials()
	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, nil, err
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

	return CallHttp(req)

}

// takes *http.Request, does not perform auth
func CallHttp(req *http.Request) (bodyMap map[string]interface{}, res *http.Response, err error) {
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println(string(reqDump)) // TODO delete
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	// resDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		return nil, nil, err
	}
	// log.Debug(string(resDump))

	body, err := ioutil.ReadAll(res.Body)


	err = json.NewDecoder(res.Body).Decode(&bodyMap)
	if err == io.EOF {
		return nil, res, nil
	} else {
		if err := json.Unmarshal([]byte(string(body)), &bodyMap); err != nil {
			return nil, nil, err
		} else {
			return bodyMap, res, nil
		}
	}
}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func PrettyPrintRequest(req *http.Request) string {
	reqDump, _ := httputil.DumpRequestOut(req, true)
	return string(reqDump)
}

func PrettyPrintResponse(res *http.Response) string {
	reqDump, _ := httputil.DumpResponse(res, true)
	return string(reqDump)
}

