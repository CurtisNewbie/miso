package config

import (
	"encoding/json"
	"fmt"
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
	RedisConf  RedisConfig  `json:"redis"`
}

// Redis configuration
type RedisConfig struct {
	Address  string `json:"address"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database int    `json:"database"`
}

// Client service configuration
type ClientConfig struct {
	// based url for file-service (should not end with '/')
	FileServiceUrl string `json:"fileServiceUrl"`

	// based url for auth-service (should not end with '/')
	AuthServiceUrl string `json:"authServiceUrl"`
}

// Database configuration
type DBConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

// Web server configuration
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// File related configuration
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

	profile = ExtractArgValue(args, func(key string) bool {
		return key == "profile"
	})

	if strings.TrimSpace(profile) == "" {
		profile = "dev" // the default is dev
	}
	return profile
}

/*
	Parse Cli Arg to extract a absolute path to the config file

	It looks for the arg that matches the pattern "configFile=[/path/to/configFile]"
	If none is found, and the profile is empty, it's by default 'app-conf-dev.json'
	If profile is specified, then it looks for 'app-conf-${profile}.json'
*/
func ParseConfigFilePath(args []string, profile string) string {
	if strings.TrimSpace(profile) == "" {
		profile = "dev"
	}

	path := ExtractArgValue(args, func(key string) bool {
		return key == "configFile"
	})

	if strings.TrimSpace(path) == "" {
		path = fmt.Sprintf("app-conf-%v.json", profile)
	}
	return path
}

/*
	Parse Cli Arg to extract a value from arg, [key]=[value]
*/
func ExtractArgValue(args []string, predicate func(key string) bool) string {
	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq != -1 {
			if key := s[:eq]; predicate(key) {
				return s[eq+1:]
			}
		}
	}
	return ""
}

// Check if it's for production by looking at the profile
func IsProd(profile string) bool {
	return profile == "prod"
}
