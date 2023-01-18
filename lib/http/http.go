package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"net/http/httputil"

	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/auth"
)

// takes as input a struct, adds auth headers
func Call(url string, method string, reqStruct interface{}, provider auth.CredentialsProvider) (map[string]interface{}, *http.Response) {
	jsonReq, err := json.Marshal(reqStruct)

	CheckOsExit(err)
	
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonReq))
	reqDump, _ := httputil.DumpRequestOut(req, true)
	log.Debug(string(reqDump))

	CheckOsExit(err)

	creds, err := provider.GetCredentials()
	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, nil // TODO better: change to return an error
	}

	if creds.Type == "BEARER" {
		req.Header.Add("Authorization", "Bearer " + creds.AccessToken)
		req.Header.Add("Content-Type", "application/json")
	} else if creds.Type == "API_KEY" {
		req.Header.Add("X-Api-Key", creds.ApiKey)
		req.Header.Add("Content-Type", "application/json")
	}
	
	return CallHttp(req)
}

// takes *http.Request, does perform auth
func CallHttpAuthed(req *http.Request, provider auth.CredentialsProvider) (bodyMap map[string]interface{}, res *http.Response) {
	creds, err := provider.GetCredentials()
	if err != nil {
		fmt.Println("Failed to get credentials")
		fmt.Println(err)
		return nil, nil // TODO better: change to return an error
	}

	if creds.Type == "BEARER" {
		log.Debug("Adding bearer token to header")
		req.Header.Add("Authorization", "Bearer " + creds.AccessToken)
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
func CallHttp(req *http.Request) (bodyMap map[string]interface{}, res *http.Response) {
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		// handle it
	}

	log.Debug(string(reqDump))
	res, _ = http.DefaultClient.Do(req)
	defer res.Body.Close()

	resDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		// do something
	}
	log.Debug(string(resDump))

	body, err := ioutil.ReadAll(res.Body)
	CheckOsExit(err)
	if err := json.Unmarshal([]byte(string(body)), &bodyMap); err != nil {
		fmt.Println(err)
		fmt.Println(string(body))
		fmt.Println(res)
		os.Exit(1)
	}
	return
	// stringBody := string(body)

	// if stringBody == "" {
	// 	return stringBody, nil, res
	// }

	// this will fail if body is nil
	// var data map[string]interface{}
	// if err := json.Unmarshal([]byte(stringBody), &data); err != nil {
	// 	fmt.Println(err)
	// 	fmt.Println(stringBody)
	// 	fmt.Println(res)
	// 	os.Exit(1)
	// }
	// return stringBody, data, res
}

func CheckOsExit(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}

func PrettyPrint(i interface{}) string {
    s, _ := json.MarshalIndent(i, "", "\t")
    return string(s)
}