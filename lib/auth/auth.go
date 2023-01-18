package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/usecakework/cakework/lib/config"
	cwConfig "github.com/usecakework/cakework/lib/config"
)

type Credentials struct {
	Type         string // either API_KEY or BEARER
	AccessToken  string // "" if type is API_KEY
	RefreshToken string // "" if type is API_KEY
	ApiKey       string // "" if type is BEARER
}

// type NoCredentialsCredentialsProvider

type CredentialsProvider interface {
	GetCredentials() (*Credentials, error)
}

// TODO rename to BearerFileCredentialsProvider
type BearerCredentialsProvider struct {
	// stuff needed for bearer tokens
	// initialize it with the logic to fetch tokens from the correct authority
	ConfigFile string
}

type BearerStringCredentialsProvider struct {
	Token string
}

// used for fly. Doesn't have a refrsh token
func (p BearerStringCredentialsProvider) GetCredentials() (*Credentials, error) {
	return &Credentials{
		Type:        "BEARER",
		AccessToken: p.Token,
	}, nil
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

	isTokenExpired, err := IsTokenExpired(config.AccessToken)
	if err != nil {
		return nil, err
	}

	if isTokenExpired {
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

	return &Credentials{
		Type:         "BEARER",
		AccessToken:  config.AccessToken,
		RefreshToken: config.RefreshToken,
		ApiKey:       "",
	}, nil
}

func IsTokenValid(token string) (bool, error) {
	AUTH0_JWKS_URL := viper.GetString("AUTH0_JWKS_URL")
	jwks, err := keyfunc.Get(AUTH0_JWKS_URL, keyfunc.Options{}) // See recommended options in the examples directory.

	parsedToken, err := jwt.Parse(token, jwks.Keyfunc)
	if err != nil {
		return false, err
	}

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

func IsTokenExpired(token string) (bool, error) {
	_, err := IsTokenValid(token)

	if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return true, nil
	}

	if err != nil {
		return true, err
	}

	return false, nil
}

// this may return status code 404 if user hasn't yet entered in the device code
func GetTokens(deviceCode string) (string, string, error) {
	AUTH0_TOKEN_URL := viper.GetString("AUTH0_TOKEN_URL")
	AUTH0_CLIENT_ID := viper.GetString("AUTH0_CLIENT_ID")
	// if using the creds to call an api, need to use the API's Identifier as the audience
	payload := strings.NewReader("grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=" + deviceCode + "&client_id=" + AUTH0_CLIENT_ID)

	req, _ := http.NewRequest("POST", AUTH0_TOKEN_URL, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	data, res, err := CallHttp(req)
	if err != nil {
		return "", "", nil
	}

	if res.StatusCode == 200 {
		log.Debug("Successfully got an access token!")
		accessToken := data["access_token"].(string)
		refreshToken := data["refresh_token"].(string)
		return accessToken, refreshToken, nil
	} else {
		return "", "", errors.New("Could not get access token, error from server " + res.Status)
	}
}

// TODO make this so that we don't update the config file
func RefreshTokens(config config.Config) (string, string, error) {
	AUTH0_CLIENT_ID := viper.GetString("AUTH0_CLIENT_ID")
	AUTH0_TOKEN_URL := viper.GetString("AUTH0_TOKEN_URL")
	// refresh the token
	payload := strings.NewReader("grant_type=refresh_token&client_id=" + AUTH0_CLIENT_ID + "&refresh_token=" + config.RefreshToken)

	req, _ := http.NewRequest("POST", AUTH0_TOKEN_URL, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	data, res, err := CallHttp(req)
	if err != nil {
		return "", "", err
	}

	if res.StatusCode == 201 || res.StatusCode == 200 /*&& strings.Contains(stringBody, "access_token")*/ {
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

func CallHttp(req *http.Request) (bodyMap map[string]interface{}, res *http.Response, err error) {
	res, _ = http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	if err := json.Unmarshal([]byte(string(body)), &bodyMap); err != nil {
		return nil, nil, err
	}
	return
}
