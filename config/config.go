package config

import (
	"encoding/json"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	// Global Configuration for the app, do not modify this
	GlobalConfig *Configuration
)

type Configuration struct {
	DBConf     DBConfig     `json:"db"`
	ServerConf ServerConfig `json:"server"`
	FileConf   FileConfig   `json:"file"`
	ClientConf ClientConfig `json:"client"`
}

type ClientConfig struct {
	// based url for file-service (should not end with '/')
	FileServiceUrl string `json:"fileServiceUrl"`

	// based url for auth-service (should not end with '/')
	AuthServiceUrl string `json:"authServiceUrl"`
}

type DBConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type FileConfig struct {
	Base string `json:"base"`
}

func SetGlobalConfig(c *Configuration) {
	GlobalConfig = c
}

/* Parse json config file */
func ParseJsonConfig(filePath string) (*Configuration, error) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Failed to open config file, %v", err)
		return nil, err
	}

	defer file.Close()

	jsonDecoder := json.NewDecoder(file)

	configuration := Configuration{}
	err = jsonDecoder.Decode(&configuration)
	if err != nil {
		log.Errorf("Failed to decode config file as json, %v", err)
		return nil, err
	}

	log.Printf("Parsed json config file: '%v'", filePath)
	return &configuration, nil
}

/*
	Parse Cli Arg to extract a profile

	It looks for the arg that matches the pattern "profile=[profileName]"
	For example, for "profile=prod", the extracted profile is "prod"
*/
func ParseProfile(args []string) string {
	profile := "dev" // the default one

	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq != -1 {
			if key := s[:eq]; key == "profile" {
				profile = s[eq+1:]
				break
			}
		}
	}

	if strings.TrimSpace(profile) == "" {
		profile = "dev" // the default is dev
	}
	return profile
}
