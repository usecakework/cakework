package http

import (
	"encoding/json"
	"net/http"

	"net/http/httputil"
)

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

