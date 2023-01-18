package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
)

var (
	tokenURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/oauth/token"
	jwksURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/.well-known/jwks.json"
)

func isTokenValid(token string) (bool, error) {
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{}) // See recommended options in the examples directory.

	parsedToken, err := jwt.Parse(token, jwks.Keyfunc)

	if parsedToken.Valid {
		// log.Debug("Token is valid")
		return true, nil
	} else if errors.Is(err, jwt.ErrTokenMalformed) {
		// log.Debug("Token is malformed")
		return false, err
	} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		// Token is either expired or not active yet
		// log.Debug("Token is expired")
		return false, err
	} else {
		// log.Debug("Couldn't handle this token:", err)
		return false, err
	}
}

func isTokenExpired(token string) bool {
	_, err := isTokenValid(token)
	if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return true
	} else {
		return false
	}
}

 // TODO hide these secrets once we make this public
 func getToken() (string, string) {
	payload := strings.NewReader("grant_type=client_credentials&client_id=" + "1iBbIn5hytDrvp4sALFMkUE49UbAC3Y0" + "&client_secret=" + "nOJkW1NOVZbxWRodmhZRun9hCCGLBsN4nAgPFeojq1W8oUJoLYTApevCaSY6wn0Q" + "&audience=https://cakework-frontend.fly.dev&scope=get:status get:result update:result update:status add:task call:task get:user submit:task create:client_token create:user get:user_from_client_token offline_access")

	req, _ := http.NewRequest("POST", tokenURL, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	stringBody, data, res := callHttp(req) // should this raise an error? 

	if (res.StatusCode == 200 || res.StatusCode == 201) && strings.Contains(stringBody, "access_token") {
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		return accessToken, refreshToken
	} else {
		return "", ""
	}
 }
func refreshTokens(refreshToken string) (string, string) {
	// refresh the token
	payload := strings.NewReader("grant_type=refresh_token&client_id=1iBbIn5hytDrvp4sALFMkUE49UbAC3Y0&refresh_token=" + refreshToken)

	req, _ := http.NewRequest("POST", tokenURL, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	stringBody, data, res := callHttp(req) // should this raise an error? 

	if (res.StatusCode == 200 || res.StatusCode == 201) && strings.Contains(stringBody, "access_token") {
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		return accessToken, refreshToken
	} else {
		return "", ""
	}
}