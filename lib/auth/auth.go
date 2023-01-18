package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/config"
	cwConfig "github.com/usecakework/cakework/lib/config"
)

var (
	tokenURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/oauth/token"
	jwksURL string = "https://dev-qanxtedlpguucmz5.us.auth0.com/.well-known/jwks.json"
)

type Credentials struct {
	Type string // either API_KEY or BEARER
	AccessToken string // "" if type is API_KEY
	RefreshToken string // "" if type is API_KEY
	ApiKey string // "" if type is BEARER
}

// type NoCredentialsCredentialsProvider

type CredentialsProvider interface {
	GetCredentials() (*Credentials, error)
}

type BearerCredentialsProvider struct {
	// stuff needed for bearer tokens
	// initialize it with the logic to fetch tokens from the correct authority
	ConfigFile string
}

func (p BearerCredentialsProvider) GetCredentials() (*Credentials, error) {
	config, err := cwConfig.LoadConfig(p.ConfigFile)
	if err != nil {
		fmt.Println("Failed to load Cakework config file")
		return nil, err
	}

	if config.AccessToken == "" || config.RefreshToken == "" {
		fmt.Println("Could not find access tokens or refresh tokens in config file")
		return nil, errors.New("Tokens are null")
	}

	if IsTokenExpired(config.AccessToken) {
		config.AccessToken, config.RefreshToken, err = RefreshTokens(*config)
		if err != nil {
			fmt.Println("Tokens expired. Failed to refresh tokens")
			return nil, err
		} else {
			if err := cwConfig.UpdateConfig(*config, p.ConfigFile); err != nil {
				fmt.Println("Failed to write refreshed tokens to config file")
				return nil, err
			}
		}
	}

	return &Credentials {
		Type: "BEARER",
		AccessToken: config.AccessToken,
		RefreshToken: config.RefreshToken,
		ApiKey: "",
	}, nil
}

/*
// TODO check if token is valid here. If it's not valid (empty or expired), fetch a token
// Q: if we tweak the config here, will it be updated in main?
func NewRequestWithAuth(method string, url string, body io.Reader, config *config.Config) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
	// config := LoadConfig()

	// Q: what if the they're logged in but neither token are valid?
	// if config.AccessToken == "" || config.RefreshToken == "" {
	// 	fmt.Println("Please sign up or log in first to get access tokens")
	// 	os.Exit(1)
	// }

	// note: this below doesn't apply for non bearer token flows!
	// TODO need to fix.
	if config.AccessToken == "" || config.RefreshToken == "" {
		fmt.Println("Please sign up or log in first to get access tokens")
		os.Exit(1)
	}
	var validToken string
	if IsTokenExpired(config.AccessToken) {
		newAccessToken, _ := RefreshTokens(config)
		validToken = newAccessToken
	} else {
		validToken = config.AccessToken
	}

    req.Header.Add("Authorization", "Bearer " + validToken)
	req.Header.Add("Content-Type", "application/json")

    return req, nil
}
*/

func IsTokenValid(token string) (bool, error) {
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

func IsTokenExpired(token string) bool {
	_, err := IsTokenValid(token)
	if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return true
	} else {
		return false
	}
}

// this may return status code 404 if user hasn't yet entered in the device code
func GetTokens(deviceCode string) (string, string) {
	// if using the creds to call an api, need to use the API's Identifier as the audience
	payload := strings.NewReader("grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=" + deviceCode + "&client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb")

	req, _ := http.NewRequest("POST", tokenURL, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

    data, res := CallHttp(req)
	
	if res.StatusCode == 200 {
		log.Debug("Successfully got an access token!")
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		return accessToken, refreshToken
	} else {
		return "", ""
	}
}

// TODO make this so that we don't update the config file
func RefreshTokens(config config.Config) (string, string, error) {
	// refresh the token
	payload := strings.NewReader("grant_type=refresh_token&client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb&refresh_token=" + config.RefreshToken)

	req, _ := http.NewRequest("POST", tokenURL, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	data, res := CallHttp(req) // should this raise an error? 

	if (res.StatusCode == 201 || res.StatusCode == 200) /*&& strings.Contains(stringBody, "access_token")*/ {
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		return accessToken, refreshToken, nil
	} else {
		fmt.Println("Got failed status code from refresh token call: ")
		fmt.Println(data)
		fmt.Println(res)
		return "", "", errors.New("Refresh token call failed")
	}
}

// copy pasta from the http package to avoid circular dependency. Think of a better way to address thi 
func CallHttp(req *http.Request) (bodyMap map[string]interface{}, res *http.Response) {
	res, _ = http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	CheckOsExit(err)
	if err := json.Unmarshal([]byte(string(body)), &bodyMap); err != nil {
		fmt.Println(err)
		fmt.Println(string(body))
		fmt.Println(res)
		os.Exit(1)
	}
	return
}

// copy pasta from the http package to avoid circular dependency. Think of a better way to address thi 
func CheckOsExit(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}