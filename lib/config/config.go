package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	UserId       string `json:"userId"`
	AccessToken  string
	RefreshToken string
	FilePath string
}

// func New(userId string, accessToken string, refreshToken string) *Config {
// 	config := &Config{
// 		UserId: userId,
// 		AccessToken: accessToken,
// 		RefreshToken: refreshToken,
// 	}

// 	return config
// }

// func (config *Config) AddConfigValue(field string, value string) error {
// 	v := reflect.ValueOf(config).Elem().FieldByName(field)
// 	if v.IsValid() {
// 		v.SetString(value)
// 	}

// 	file, _ := json.MarshalIndent(config, "", " ")

// 	if err := ioutil.WriteFile(config.FilePath, file, 0644); err != nil {
// 		return err
// 	} 
// 	return nil
// }

func LoadConfig(configPath string) (*Config, error) {
	var config Config
	var jsonFile *os.File
	if _, err := os.Stat(configPath); err == nil {
		jsonFile, err = os.Open(configPath)
	} else { // assume that error is because file doesn't exist
		jsonFile, err = os.Create(configPath)
		if err != nil {
			return nil, err
		}
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(byteValue, &config)
	return &config, nil
}

func UpdateConfig(config Config, configPath string) error {
	file, _ := json.MarshalIndent(config, "", " ")

	if err := ioutil.WriteFile(configPath, file, 0644); err != nil {
		return err
	} 
	return nil
}