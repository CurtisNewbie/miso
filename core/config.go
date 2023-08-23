package core

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	// regex for arg expansion
	resolveArgRegexp = regexp.MustCompile(`\${[a-zA-Z0-9\\-\\_\.]+}`)

	// mutex for viper
	viperRWMutex sync.RWMutex

	PRODUCTION_PROFILE_NAME = "prod"
)

func init() {
	SetDefProp(PROP_PROFILE, "dev")
	SetDefProp(PROP_PRODUCTION_MODE, "false")
}

// Set default value for the prop
func SetProp(prop string, val any) {
	doWithViperWriteLock(func() { viper.Set(prop, val) })
}

// Set default value for the prop
func SetDefProp(prop string, defVal any) {
	doWithViperWriteLock(func() { viper.SetDefault(prop, defVal) })
}

// Check whether the prop exists
func ContainsProp(prop string) bool {
	return HasProp(prop)
}

// Check whether the prop exists
func HasProp(prop string) bool {
	return doWithViperReadLock(func() bool { return viper.IsSet(prop) })
}

// Get prop as int slice
func GetConfIntSlice(prop string) []int {
	return GetPropIntSlice(prop)
}

// Get prop as int slice
func GetPropIntSlice(prop string) []int {
	return doWithViperReadLock(func() []int { return viper.GetIntSlice(prop) })
}

// Get prop as string slice
func GetPropStringSlice(prop string) []string {
	return doWithViperReadLock(func() []string { return viper.GetStringSlice(prop) })
}

// Get prop as int
func GetPropInt(prop string) int {
	return doWithViperReadLock(func() int { return viper.GetInt(prop) })
}

// Get prop as bool
func GetPropBool(prop string) bool {
	return doWithViperReadLock(func() bool { return viper.GetBool(prop) })
}

/*
Get prop as string

If the value is an argument that can be expanded, the actual value will be resolved if possible.

e.g, for "name" : "${secretName}".

This func will attempt to resolve the actual value for '${secretName}'.
*/
func GetPropStr(prop string) string {
	return ResolveArg(_getPropString(prop))
}

// Get prop as string (with lock)
func _getPropString(prop string) string {
	return doWithViperReadLock(func() string { return viper.GetString(prop) })
}

// Unmarshal to object from properties
func UnmarshalFromProp(ptr any) {
	doWithViperReadLock(func() any {
		viper.Unmarshal(ptr)
		return nil
	})
}

/*
Default way to read config file.

By reading the provided args, this func identifies the profile to use and the
associated name of the config file to look for.

Repetitively calling this method overides previously loaded config.

You can also use ReadConfig to load your custom configFile.

It's essentially:

	LoadConfigFromFile(GuessConfigFilePath(args, GuessProfile(args)))

Notice that the loaded configuration can be overriden by the cli arguments as well by using `KEY=VALUE` syntax.
*/
func DefaultReadConfig(args []string, c Rail) {
	profile := GuessProfile(args)
	SetProfile(profile)

	if strings.ToLower(profile) == PRODUCTION_PROFILE_NAME {
		SetProp(PROP_PRODUCTION_MODE, true)
	}

	configFile := GuessConfigFilePath(args, profile)
	LoadConfigFromFile(configFile, c)

	// overwrite loaded configuration with environment variables
	env := os.Environ()
	kv := ArgKeyVal(env)
	for k, v := range kv {
		SetProp(k, v)
	}

	// overwrite the loaded configuration with cli arguments
	kv = ArgKeyVal(args)
	for k, v := range kv {
		SetProp(k, v)
	}
}

/*
Load config from file

Repetitively calling this method overides previously loaded config.
*/
func LoadConfigFromFile(configFile string, r Rail) {
	if configFile == "" {
		return
	}

	loaded := false
	doWithViperWriteLock(func() {
		f, err := os.Open(configFile)
		if err != nil {
			if os.IsNotExist(err) {
				r.Debugf("Unable to find config file: '%s'", configFile)
				return
			}
			r.Fatalf("Failed to open config file: '%s', %v", configFile, err)
		}
		viper.SetConfigType("yml")
		if err = viper.ReadConfig(bufio.NewReader(f)); err != nil {
			r.Fatalf("Failed to load config file: '%s', %v", configFile, err)
		}
		loaded = true
	})

	if loaded {
		r.Debugf("Loaded config file: '%v'", configFile)
	}
}

// Get profile
func GetProfile() (profile string) {
	profile = GetPropStr(PROP_PROFILE)
	return
}

// Set profile
func SetProfile(profile string) {
	SetProp(PROP_PROFILE, profile)
}

/*
Parse Cli Arg to extract a profile

It looks for the arg that matches the pattern "profile=[profileName]"
For example, for "profile=prod", the extracted profile is "prod"
*/
func GuessProfile(args []string) string {
	profile := "dev" // the default one

	profile = ExtractArgValue(args, func(key string) bool { return key == PROP_PROFILE })
	if strings.TrimSpace(profile) == "" {
		profile = "dev" // the default is dev
	}
	return profile
}

/*
Parse args to guess a absolute path to the config file

- It looks for the arg that matches the pattern "configFile=/path/to/configFile".

- If none is found, and the profile is empty, it's by default 'app-conf-dev.yml'.

- If profile is specified, then it looks for 'app-conf-${profile}.yml'.
*/
func GuessConfigFilePath(args []string, profile string) string {
	if strings.TrimSpace(profile) == "" {
		profile = "dev"
	}

	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" })
	if strings.TrimSpace(path) == "" {
		path = fmt.Sprintf("app-conf-%v.yml", profile)
	}
	return path
}

/*
Parse CLI Arg to extract a value from arg, [key]=[value]

e.g.,

To look for 'configFile=?'.

	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" }).
*/
func ExtractArgValue(args []string, predicate Predicate[string]) string {
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

/*
Parse CLI args to key-value map
*/
func ArgKeyVal(args []string) map[string]string {
	m := map[string]string{}
	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq == -1 {
			continue
		}

		key := strings.TrimSpace(s[:eq])
		val := strings.TrimSpace(s[eq+1:])
		m[key] = val
	}
	return m
}

// Get environment variable
func GetEnv(key string) string {
	return os.Getenv(key)
}

// Set environment variable
func SetEnv(key string, val string) {
	os.Setenv(key, val)
}

// Get environment variable with default value
func GetEnvElse(key string, defVal string) string {
	s := GetEnv(key)
	if s == "" {
		return defVal
	}
	return s
}

// Check whether we are running in production mode
//
// This func looks for prop: PROP_PRODUCTION_MODE,
// if the prop value equals to true (case insensitive), then
// true is returned else false
func IsProdMode() bool {
	if !ContainsProp(PROP_PRODUCTION_MODE) {
		return false
	}
	mode := GetPropBool(PROP_PRODUCTION_MODE)
	return mode
}

// Resolve server host, use IPV4 if the given address is empty or '0.0.0.0'
func ResolveServerHost(address string) string {
	if IsBlankStr(address) || address == LOCAL_IP_ANY {
		address = GetLocalIPV4()
	}
	return address
}

// Resolve argument, e.g., for arg like '${someArg}', it will in fact look for 'someArg' in os.Env
func ResolveArg(arg string) string {
	return resolveArgRegexp.ReplaceAllStringFunc(arg, func(s string) string {
		r := []rune(s)
		key := string(r[2 : len(r)-1])
		val := GetEnv(key)

		if val == "" {
			val = GetPropStr(key)
		}

		if val == "" {
			val = s
		}
		return val
	})
}

// call with viper lock
func doWithViperWriteLock(f func()) {
	viperRWMutex.Lock()
	defer viperRWMutex.Unlock()
	f()
}

// call and return with viper lock
func doWithViperReadLock[T any](f func() T) T {
	viperRWMutex.RLock()
	defer viperRWMutex.RUnlock()
	return f()
}
