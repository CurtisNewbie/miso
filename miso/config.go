package miso

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/spf13/viper"
)

var (
	// regex for arg expansion
	resolveArgRegexp = regexp.MustCompile(`\${[a-zA-Z0-9\\-\\_\.]+}`)

	// mutex for viper
	viperRWMutex sync.RWMutex

	// var for config.go
	configVar = configVarHolder{
		fastBoolCache: make(map[string]bool),
	}
)

type configVarHolder struct {
	sync.RWMutex
	// fast bool cache, GetBool() is a frequent operation, this aims to speed up the key lookup.
	fastBoolCache map[string]bool
}

func init() {
	SetDefProp(PropProdMode, true)
}

// Set value for the prop
func SetProp(prop string, val any) {
	doWithViperWriteLock(func() {
		cleanFastBoolCache(prop)
		viper.Set(prop, val)
	})
}

// Set default value for the prop
func SetDefProp(prop string, defVal any) {
	doWithViperWriteLock(func() {
		cleanFastBoolCache(prop)
		viper.SetDefault(prop, defVal)
	})
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
func GetPropStrSlice(prop string) []string {
	return doWithViperReadLock(func() []string { return viper.GetStringSlice(prop) })
}

// Get prop as int
func GetPropInt(prop string) int {
	return doWithViperReadLock(func() int { return viper.GetInt(prop) })
}

// Get prop as string based map.
func GetPropStrMap(prop string) map[string]string {
	return doWithViperReadLock(func() map[string]string { return viper.GetStringMapString(prop) })
}

// Get prop as time.Duration
func GetPropDur(prop string, unit time.Duration) time.Duration {
	return time.Duration(GetPropInt(prop)) * unit
}

// Get prop as bool
func GetPropBool(prop string) bool {
	return doWithViperReadLock(func() bool {
		configVar.RLock()
		v, ok := configVar.fastBoolCache[prop]
		if ok {
			configVar.RUnlock()
			return v
		}
		configVar.RUnlock()

		v = viper.GetBool(prop)

		configVar.Lock()
		defer configVar.Unlock()
		configVar.fastBoolCache[prop] = v
		return v
	})
}

// clean the fast bool cache
func cleanFastBoolCache(prop string) {
	configVar.Lock()
	defer configVar.Unlock()
	delete(configVar.fastBoolCache, prop)
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

// Unmarshal configuration.
func UnmarshalFromProp(ptr any) {
	doWithViperReadLock(func() any {
		if err := viper.Unmarshal(ptr); err != nil {
			Warnf("failed to UnmarshalFromProp, %v", err)
		}
		return nil
	})
}

// Unmarshal configuration from a speicific key.
func UnmarshalFromPropKey(key string, ptr any) {
	doWithViperReadLock(func() any {
		if err := viper.UnmarshalKey(key, ptr); err != nil {
			Warnf("failed to UnmarshalFromPropKey, %v", err)
		}
		return nil
	})
}

// Overwrite existing conf using environment and cli args.
func OverwriteConf(args []string) {
	// overwrite loaded configuration with environment variables
	overwriteConf(ArgKeyVal(os.Environ()))
	// overwrite the loaded configuration with cli arguments
	overwriteConf(ArgKeyVal(args))
}

/*
Default way to read config file.

Repetitively calling this method overides previously loaded config.

You can also use ReadConfig to load your custom configFile. This func is essentially:

	LoadConfigFromFile(GuessConfigFilePath(args))

Notice that the loaded configuration can be overriden by the cli arguments as well by using `KEY=VALUE` syntax.
*/
func DefaultReadConfig(args []string, rail Rail) {
	loaded := util.NewSet[string]()

	defConfigFile := GuessConfigFilePath(args)
	loaded.Add(defConfigFile)

	if err := LoadConfigFromFile(defConfigFile, rail); err != nil {
		rail.Debugf("Failed to load config file, file: %v, %v", defConfigFile, err)
	} else {
		rail.Infof("Loaded config file: %v", defConfigFile)
	}

	// the load config file may specifiy extra files to be loaded
	extraFiles := GetPropStrSlice(PropConfigExtraFiles)

	for i := range extraFiles {
		f := extraFiles[i]

		if !loaded.Add(f) {
			continue
		}

		if ok, err := util.FileExists(f); err != nil || !ok {
			if err != nil {
				rail.Warnf("Failed to open extra config file, %v, %v", f, err)
			}

			rail.Debugf("Extra config file %v not found", f)
			continue
		}

		if err := LoadConfigFromFile(f, rail); err != nil {
			rail.Warnf("Failed to load extra config file, %v, %v", f, err)
		} else {
			rail.Infof("Loaded config file: %v", f)
		}
	}

	OverwriteConf(args)

	// try again, one may specify the extra files through cli args or environment variables
	extraFiles = GetPropStrSlice(PropConfigExtraFiles)
	for i := range extraFiles {
		f := extraFiles[i]
		if !loaded.Add(f) {
			continue
		}

		if err := LoadConfigFromFile(f, rail); err != nil {
			rail.Warnf("Failed to load extra config file, %v, %v", f, err)
		} else {
			rail.Infof("Loaded extra config file: %v", f)
		}
	}
}

// Load config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method overides previously loaded config.
func LoadConfigFromReader(reader io.Reader, r Rail) error {
	var eo error

	doWithViperWriteLock(func() {
		viper.SetConfigType("yml")
		if err := viper.MergeConfig(reader); err != nil {
			eo = fmt.Errorf("failed to load config from reader: %v", err)
		}

		// reset the whole fastBoolCache
		configVar.Lock()
		defer configVar.Unlock()
		if len(configVar.fastBoolCache) > 0 {
			configVar.fastBoolCache = make(map[string]bool)
		}
	})

	return eo
}

// Load config from string.
//
// Calling this method overides previously loaded config.
func LoadConfigFromStr(s string, r Rail) error {
	sr := bytes.NewReader(util.UnsafeStr2Byt(s))
	return LoadConfigFromReader(sr, r)
}

// Load config from file.
//
// Calling this method overides previously loaded config.
func LoadConfigFromFile(configFile string, r Rail) error {
	if configFile == "" {
		return nil
	}

	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unable to find config file: '%s'", configFile)
		}
		return fmt.Errorf("failed to open config file: '%s', %v", configFile, err)
	}
	defer f.Close()

	err = LoadConfigFromReader(f, r)
	if err != nil {
		return fmt.Errorf("failed to load config file: '%s', %v", configFile, err)
	}
	r.Debugf("Loaded config file: '%v'", configFile)
	return nil
}

// Guess config file path.
//
// It first looks for the arg that matches the pattern "configFile=/path/to/configFile".
// If none is found, it's by default 'conf.yml'.
func GuessConfigFilePath(args []string) string {
	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" })
	if strings.TrimSpace(path) == "" {
		path = "conf.yml"
	}
	return path
}

/*
Parse CLI Arg to extract a value from arg, [key]=[value]

e.g.,

To look for 'configFile=?'.

	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" }).
*/
func ExtractArgValue(args []string, predicate util.Predicate[string]) string {
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

func overwriteConf(kvs map[string][]string) {
	for k, v := range kvs {
		if len(v) == 1 {
			SetProp(k, v[0])
		} else {
			SetProp(k, v)
		}
	}
}

// Parse CLI args to key-value map
func ArgKeyVal(args []string) map[string][]string {
	m := map[string][]string{}
	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq == -1 {
			continue
		}

		key := strings.TrimSpace(s[:eq])
		val := strings.TrimSpace(s[eq+1:])
		if prev, ok := m[key]; ok {
			m[key] = append(prev, val)
		} else {
			m[key] = []string{val}
		}
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
func IsProdMode() bool {
	return GetPropBool(PropProdMode)
}

// Resolve server host, use IPV4 if the given address is empty or '0.0.0.0'
func ResolveServerHost(address string) string {
	if util.IsBlankStr(address) || address == util.LocalIpAny {
		address = util.GetLocalIPV4()
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
