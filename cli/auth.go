package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
)

var (
	tokenURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/oauth/token"
	jwksURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/.well-known/jwks.json"
)

// TODO check if token is valid here. If it's not valid (empty or expired), fetch a token
// Q: if we tweak the config here, will it be updated in main?
func newRequestWithAuth(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
	config := loadConfig()

	// Q: what if the they're logged in but neither token are valid?
	if config.AccessToken == "" || config.RefreshToken == "" {
		fmt.Println("Please sign up or log in first to get access tokens")
		os.Exit(1)
	}
	var validToken string
	if isTokenExpired(config.AccessToken) {
		newAccessToken, _ := refreshTokens(config.AccessToken, config.RefreshToken)
		validToken = newAccessToken
	} else {
		validToken = config.AccessToken
	}

    req.Header.Add("Authorization", "Bearer " + validToken)
	req.Header.Add("Content-Type", "application/json")

    return req, nil
}

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

// this may return status code 404 if user hasn't yet entered in the device code
func getTokens(deviceCode string) (string, string) {
	// if using the creds to call an api, need to use the API's Identifier as the audience
	payload := strings.NewReader("grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=" + deviceCode + "&client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb")

	req, _ := http.NewRequest("POST", tokenURL, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

    bodyString, data, res := callHttp(req)
	
	if res.StatusCode == 200 && strings.Contains(bodyString, "access_token") {
			log.Debug("Successfully got an access token!")
			accessToken := data["access_token"].(string)
			refreshToken := data["refresh_token"].(string)
			return accessToken, refreshToken
	} else {
		return "", ""
	}
}

func refreshTokens(token string, refreshToken string) (string, string) {
	// refresh the token
	payload := strings.NewReader("grant_type=refresh_token&client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb&refresh_token=" + config.RefreshToken)

	req, _ := http.NewRequest("POST", tokenURL, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	stringBody, data, res := callHttp(req) // should this raise an error? 

	if (res.StatusCode == 201 || res.StatusCode == 201) && strings.Contains(stringBody, "access_token") {
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		addConfigValue("AccessToken", accessToken)
		addConfigValue("RefreshToken", refreshToken)
		return accessToken, refreshToken
	} else {
		return "", ""
	}
}