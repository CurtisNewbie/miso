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

	setDefPropFuncs []func() (k string, defVal any)
)

func init() {
	SetDefProp(PropProdMode, true)
}

type AppConfig struct {
	vp   *viper.Viper
	rwmu *sync.RWMutex

	// fast bool cache, GetBool() is a frequent operation, this aims to speed up the key lookup.
	fastBoolCache *util.RWMap[string, bool]
}

func (a *AppConfig) _appConfigDoWithWLock(f func()) {
	a.rwmu.Lock()
	defer a.rwmu.Unlock()
	f()
}

func (a *AppConfig) _appConfigDoWithRLock(f func() any) any {
	a.rwmu.RLock()
	defer a.rwmu.RUnlock()
	return f()
}

// Set value for the prop
func (a *AppConfig) SetProp(prop string, val any) {
	doWithWriteLock(a, func() {
		a.fastBoolCache.Del(prop)
		a.vp.Set(prop, val)
	})
}

// Set default value for the prop
func (a *AppConfig) SetDefProp(prop string, defVal any) {
	doWithWriteLock(a, func() {
		a.fastBoolCache.Del(prop)
		a.vp.SetDefault(prop, defVal)
	})
}

// Check whether the prop exists
func (a *AppConfig) HasProp(prop string) bool {
	return returnWithReadLock(a, func() bool { return a.vp.IsSet(prop) })
}

// Get prop as int slice
func (a *AppConfig) GetPropIntSlice(prop string) []int {
	return returnWithReadLock(a, func() []int { return a.vp.GetIntSlice(prop) })
}

// Get prop as string slice
func (a *AppConfig) GetPropStrSlice(prop string) []string {
	return returnWithReadLock(a, func() []string { return a.vp.GetStringSlice(prop) })
}

// Get prop as int
func (a *AppConfig) GetPropInt(prop string) int {
	return returnWithReadLock(a, func() int { return a.vp.GetInt(prop) })
}

// Get prop as string based map.
func (a *AppConfig) GetPropStrMap(prop string) map[string]string {
	return returnWithReadLock(a, func() map[string]string { return a.vp.GetStringMapString(prop) })
}

// Get prop as time.Duration
func (a *AppConfig) GetPropDur(prop string, unit time.Duration) time.Duration {
	return time.Duration(a.GetPropInt(prop)) * unit
}

// Get prop as bool
func (a *AppConfig) GetPropBool(prop string) bool {
	return returnWithReadLock(a, func() bool {
		v, _ := a.fastBoolCache.GetElse(prop, func(k string) bool {
			return a.vp.GetBool(k)
		})
		return v
	})
}

/*
Get prop as string

If the value is an argument that can be expanded, the actual value will be resolved if possible.

e.g, for "name" : "${secretName}".

This func will attempt to resolve the actual value for '${secretName}'.
*/
func (a *AppConfig) GetPropStr(prop string) string {
	return a.ResolveArg(returnWithReadLock(a, func() string { return a.vp.GetString(prop) }))
}

// Unmarshal configuration.
func (a *AppConfig) UnmarshalFromProp(ptr any) {
	doWithReadLock(a, func() {
		if err := a.vp.Unmarshal(ptr); err != nil {
			Warnf("failed to UnmarshalFromProp, %v", err)
		}
	})
}

// Unmarshal configuration from a speicific key.
func (a *AppConfig) UnmarshalFromPropKey(key string, ptr any) {
	doWithReadLock(a, func() {
		if err := a.vp.UnmarshalKey(key, ptr); err != nil {
			Warnf("failed to UnmarshalFromPropKey, %v", err)
		}
	})
}

// Overwrite existing conf using environment and cli args.
func (a *AppConfig) OverwriteConf(args []string) {
	// overwrite loaded configuration with environment variables
	a.overwriteConf(ArgKeyVal(os.Environ()))
	// overwrite the loaded configuration with cli arguments
	a.overwriteConf(ArgKeyVal(args))
}

/*
Default way to read config file.

Repetitively calling this method overides previously loaded config.

You can also use ReadConfig to load your custom configFile. This func is essentially:

	LoadConfigFromFile(GuessConfigFilePath(args))

Notice that the loaded configuration can be overriden by the cli arguments as well by using `KEY=VALUE` syntax.
*/
func (a *AppConfig) DefaultReadConfig(args []string) {
	loaded := util.NewSet[string]()

	defConfigFile := GuessConfigFilePath(args)
	loaded.Add(defConfigFile)

	if err := a.LoadConfigFromFile(defConfigFile); err != nil {
		Debugf("Failed to load config file, file: %v, %v", defConfigFile, err)
	} else {
		Infof("Loaded config file: %v", defConfigFile)
	}

	// the load config file may specifiy extra files to be loaded
	extraFiles := a.GetPropStrSlice(PropConfigExtraFiles)

	for i := range extraFiles {
		f := extraFiles[i]

		if !loaded.Add(f) {
			continue
		}

		if ok, err := util.FileExists(f); err != nil || !ok {
			if err != nil {
				Warnf("Failed to open extra config file, %v, %v", f, err)
			}

			Debugf("Extra config file %v not found", f)
			continue
		}

		if err := a.LoadConfigFromFile(f); err != nil {
			Warnf("Failed to load extra config file, %v, %v", f, err)
		} else {
			Infof("Loaded config file: %v", f)
		}
	}

	a.OverwriteConf(args)

	// try again, one may specify the extra files through cli args or environment variables
	extraFiles = a.GetPropStrSlice(PropConfigExtraFiles)
	for i := range extraFiles {
		f := extraFiles[i]
		if !loaded.Add(f) {
			continue
		}

		if err := a.LoadConfigFromFile(f); err != nil {
			Warnf("Failed to load extra config file, %v, %v", f, err)
		} else {
			Infof("Loaded extra config file: %v", f)
		}
	}
}

// Load config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method overides previously loaded config.
func (a *AppConfig) LoadConfigFromReader(reader io.Reader) error {
	var eo error

	doWithWriteLock(a, func() {
		a.vp.SetConfigType("yml")
		if err := a.vp.MergeConfig(reader); err != nil {
			eo = fmt.Errorf("failed to load config from reader: %v", err)
		}

		// reset the whole fastBoolCache
		a.fastBoolCache.Clear()
	})

	return eo
}

// Load config from string.
//
// Calling this method overides previously loaded config.
func (a *AppConfig) LoadConfigFromStr(s string) error {
	sr := bytes.NewReader(util.UnsafeStr2Byt(s))
	return a.LoadConfigFromReader(sr)
}

// Load config from file.
//
// Calling this method overides previously loaded config.
func (a *AppConfig) LoadConfigFromFile(configFile string) error {
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

	err = a.LoadConfigFromReader(f)
	if err != nil {
		return fmt.Errorf("failed to load config file: '%s', %v", configFile, err)
	}
	Debugf("Loaded config file: '%v'", configFile)
	return nil
}

func (a *AppConfig) overwriteConf(kvs map[string][]string) {
	for k, v := range kvs {
		if len(v) == 1 {
			a.SetProp(k, v[0])
		} else {
			a.SetProp(k, v)
		}
	}
}

// Check whether we are running in production mode
func (a *AppConfig) IsProdMode() bool {
	return a.GetPropBool(PropProdMode)
}

// Resolve argument, e.g., for arg like '${someArg}', it will in fact look for 'someArg' in os.Env
func (a *AppConfig) ResolveArg(arg string) string {
	return resolveArgRegexp.ReplaceAllStringFunc(arg, func(s string) string {
		r := []rune(s)
		key := string(r[2 : len(r)-1])
		val := GetEnv(key)

		if val == "" {
			val = a.GetPropStr(key)
		}

		if val == "" {
			val = s
		}
		return val
	})
}

func newAppConfig() *AppConfig {
	return &AppConfig{
		vp:            viper.New(),
		rwmu:          &sync.RWMutex{},
		fastBoolCache: util.NewRWMap[string, bool](),
	}
}

// Set value for the prop
func SetProp(prop string, val any) {
	globalConfig().SetProp(prop, val)
}

// Set default value for the prop
func SetDefProp(prop string, defVal any) {
	setDefPropFuncs = append(setDefPropFuncs, func() (string, any) { return prop, defVal })
}

func runSetDefPropFuncs(app *MisoApp) {
	for _, f := range setDefPropFuncs {
		app.config.SetDefProp(f())
	}
}

// Check whether the prop exists
//
// deprecated: use HasProp(..) instead.
func ContainsProp(prop string) bool {
	return globalConfig().HasProp(prop)
}

// Check whether the prop exists
func HasProp(prop string) bool {
	return globalConfig().HasProp(prop)
}

// Get prop as int slice
//
// deprecated: use GetPropIntSlice(..) instead.
func GetConfIntSlice(prop string) []int {
	return globalConfig().GetPropIntSlice(prop)
}

// Get prop as int slice
func GetPropIntSlice(prop string) []int {
	return globalConfig().GetPropIntSlice(prop)
}

// Get prop as string slice
func GetPropStrSlice(prop string) []string {
	return globalConfig().GetPropStrSlice(prop)
}

// Get prop as int
func GetPropInt(prop string) int {
	return globalConfig().GetPropInt(prop)
}

// Get prop as string based map.
func GetPropStrMap(prop string) map[string]string {
	return globalConfig().GetPropStrMap(prop)
}

// Get prop as time.Duration
func GetPropDur(prop string, unit time.Duration) time.Duration {
	return globalConfig().GetPropDur(prop, unit)
}

// Get prop as bool
func GetPropBool(prop string) bool {
	return globalConfig().GetPropBool(prop)
}

/*
Get prop as string

If the value is an argument that can be expanded, the actual value will be resolved if possible.

e.g, for "name" : "${secretName}".

This func will attempt to resolve the actual value for '${secretName}'.
*/
func GetPropStr(prop string) string {
	return globalConfig().GetPropStr(prop)
}

// Unmarshal configuration.
func UnmarshalFromProp(ptr any) {
	globalConfig().UnmarshalFromProp(ptr)
}

// Unmarshal configuration from a speicific key.
func UnmarshalFromPropKey(key string, ptr any) {
	globalConfig().UnmarshalFromPropKey(key, ptr)
}

// Overwrite existing conf using environment and cli args.
func OverwriteConf(args []string) {
	globalConfig().OverwriteConf(args)
}

/*
Default way to read config file.

Repetitively calling this method overides previously loaded config.

You can also use ReadConfig to load your custom configFile. This func is essentially:

	LoadConfigFromFile(GuessConfigFilePath(args))

Notice that the loaded configuration can be overriden by the cli arguments as well by using `KEY=VALUE` syntax.
*/
func DefaultReadConfig(args []string, rail Rail) {
	globalConfig().DefaultReadConfig(args)
}

// Load config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method overides previously loaded config.
func LoadConfigFromReader(reader io.Reader, r Rail) error {
	return globalConfig().LoadConfigFromReader(reader)
}

// Load config from string.
//
// Calling this method overides previously loaded config.
func LoadConfigFromStr(s string, r Rail) error {
	return globalConfig().LoadConfigFromStr(s)
}

// Load config from file.
//
// Calling this method overides previously loaded config.
func LoadConfigFromFile(configFile string, r Rail) error {
	return globalConfig().LoadConfigFromFile(configFile)
}

// Check whether we are running in production mode
func IsProdMode() bool {
	return globalConfig().IsProdMode()
}

// Resolve '${someArg}' style variables.
func ResolveArg(arg string) string {
	return globalConfig().ResolveArg(arg)
}

// call with viper lock
func doWithWriteLock(a *AppConfig, f func()) {
	a._appConfigDoWithWLock(func() {
		f()
	})
}

func returnWithReadLock[T any](a *AppConfig, f func() T) T {
	v := a._appConfigDoWithRLock(func() any {
		return f()
	})
	if v == nil {
		var t T
		return t
	}
	return v.(T)
}

func doWithReadLock(a *AppConfig, f func()) {
	a._appConfigDoWithRLock(func() any {
		f()
		return nil
	})
}

// -------------

// Resolve server host, use IPV4 if the given address is empty or '0.0.0.0'
func ResolveServerHost(address string) string {
	if util.IsBlankStr(address) || address == util.LocalIpAny {
		address = util.GetLocalIPV4()
	}
	return address
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

func globalConfig() *AppConfig {
	return App().config
}
