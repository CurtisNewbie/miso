package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/curtisnewbie/gocommon/util"
	"github.com/sirupsen/logrus"
)

var (
	// Global Configuration for the app, do not modify this
	GlobalConfig *Configuration
	isProd       bool = false
	isProdLock   sync.RWMutex
)

type Configuration struct {
	ServerConf ServerConfig  `json:"server"`
	FileConf   FileConfig    `json:"file"`
	DBConf     *DBConfig     `json:"db"`
	ClientConf *ClientConfig `json:"client"`
	RedisConf  *RedisConfig  `json:"redis"`
	ConsulConf *ConsulConfig `json:"consul"`
}

// Redis configuration
type RedisConfig struct {
	Enabled  bool   `json:"enabled"`
	Address  string `json:"address"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database int    `json:"database"`
}

func (pt *RedisConfig) String() string {
	return fmt.Sprintf("{Enabled:%t Address:%s, Port:%s, Username:%s, Password:***, Database:%d}", pt.Enabled, pt.Address,
		pt.Port, pt.Username, pt.Database)
}

// Client service configuration
type ClientConfig struct {
	// based url for file-service (should not end with '/')
	FileServiceUrl string `json:"fileServiceUrl"`

	// based url for auth-service (should not end with '/')
	AuthServiceUrl string `json:"authServiceUrl"`
}

func (pt *ClientConfig) String() string {
	return fmt.Sprintf("{FileServiceUrl:%s AuthServiceUrl:%s}", pt.FileServiceUrl, pt.AuthServiceUrl)
}

// Database configuration
type DBConfig struct {
	Enabled  bool   `json:"enabled"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

func (pt *DBConfig) String() string {
	return fmt.Sprintf("{Enabled:%t User:%s Password:*** Database:%s Host:%s Port:%s}", pt.Enabled, pt.User, pt.Database, pt.Host, pt.Port)
}

// Web server configuration
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// File related configuration
type FileConfig struct {
	Base string `json:"base"`
	Temp string `json:"temp"`
}

// Consul configuration
type ConsulConfig struct {
	Enabled             bool   `json:"enabled"`
	RegisterName        string `json:"registerName"`
	RegisterAddress     string `json:"registerAddress"`
	ConsulAddress       string `json:"consulAddress"`
	HealthCheckUrl      string `json:"healthCheckUrl"`
	HealthCheckInterval string `json:"healthCheckInterval"`
	HealthCheckTimeout  string `json:"healthCheckTimeout"`
}

func (pt *ConsulConfig) String() string {
	return fmt.Sprintf("{Enabled:%t RegisterName:%s, ConsulAddress:%s, HealthCheckUrl:%s, HealthCheckInterval:%s, HealthCheckTimeout:%s}",
		pt.Enabled, pt.RegisterName, pt.ConsulAddress, pt.HealthCheckUrl, pt.HealthCheckInterval, pt.HealthCheckTimeout)
}

// Set the globalConfig
func SetGlobalConfig(c *Configuration) {
	GlobalConfig = c
}

/*
	Default way to parse profile and configuration from os.Args, panic if failed

	Apart from read the json config file, it also does extra configuration, e.g.,
	resolving ServerConf.Host if absent, generating healthCheckUrl if necessary and so on
*/
func DefaultParseProfConf(args []string) (profile string, conf *Configuration) {
	profile = ParseProfile(args)
	logrus.Printf("Using profile: '%v'", profile)

	configFile := ParseConfigFilePath(args[1:], profile)
	logrus.Printf("Looking for config file: '%v'", configFile)

	conf, err := ParseJsonConfig(configFile)
	if err != nil {
		panic(err)
	}

	// use the ipv4 extract from net interface unless localhost or 127.0.0.1 is specified
	serverHost := conf.ServerConf.Host

	if conf.ConsulConf != nil {

		// default health endpoint (/health)
		if conf.ConsulConf.HealthCheckUrl == "" {
			conf.ConsulConf.HealthCheckUrl = "/health"
			logrus.Infof("Using default health check endpoint: '%s'", conf.ConsulConf.HealthCheckUrl)
		}

		// default address used to register on consul
		if conf.ConsulConf.RegisterAddress == "" {
			conf.ConsulConf.RegisterAddress = ResolveServerHost(serverHost)
			logrus.Infof("Will register on Consul using address: '%s'", conf.ConsulConf.RegisterAddress)
		}
	}

	SetGlobalConfig(conf)
	SetIsProdMode(IsProd(profile))
	logrus.Infof("Using conf file: %+v", *conf)
	return
}

/* Parse json config file */
func ParseJsonConfig(filePath string) (*Configuration, error) {

	file, err := os.Open(filePath)
	if err != nil {
		logrus.Errorf("Failed to open config file, %v", err)
		return nil, err
	}

	defer file.Close()

	jsonDecoder := json.NewDecoder(file)

	configuration := Configuration{}
	err = jsonDecoder.Decode(&configuration)
	if err != nil {
		logrus.Errorf("Failed to decode config file as json, %v", err)
		return nil, err
	}

	logrus.Printf("Parsed json config file: '%v'", filePath)
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

// Get environment variable
func GetEnv(key string) string {
	return os.Getenv(key)
}

// Get environment variable with default value
func GetEnvElse(key string, defVal string) string {
	s := GetEnv(key)
	if s == "" {
		return defVal
	}
	return s
}

// mark that we are running in production mode
func SetIsProdMode(isProdFlag bool) {
	isProdLock.Lock()
	defer isProdLock.Unlock()
	isProd = isProdFlag
}

// check whether we are running in production mode
func IsProdMode() bool {
	isProdLock.RLock()
	defer isProdLock.RUnlock()
	return isProd
}

// Resolve server host, use IPV4 if the given address is empty or '0.0.0.0'
func ResolveServerHost(address string) string {
	if util.IsStrEmpty(address) || address == util.LOCAL_IP_ANY {
		address = util.GetLocalIPV4()
	}
	return address
}
