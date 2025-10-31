package miso

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	defConfigFilename = "conf.yml"

	// regex for arg expansion
	resolveArgRegexp = regexp.MustCompile(`\${[a-zA-Z0-9\/\-\_\.: ]+}`)
)

type AppConfig struct {
	vp   *viper.Viper
	rwmu *sync.RWMutex

	defaultConfigFileLoaded []string

	// fast bool cache, GetBool() is a frequent operation, this aims to speed up the key lookup.
	// key is always the real key not the alias
	fastBoolCache *hash.StrRWMap[bool]

	// aliases of keys
	// alias -> key
	// aliases map[string]string
}

func (a *AppConfig) WriteConfigAs(filename string) (err error) {
	a._appConfigDoWithRLock(func() any {
		err = a.vp.WriteConfigAs(filename)
		return nil
	})
	return
}

func (a *AppConfig) aliasLookup(k string) string {
	// newkey, exists := a.aliases[k]
	// if exists {
	// 	return a.aliasLookup(newkey)
	// }
	return k
}

/*
func (a *AppConfig) RegisterAlias(alias, key string) {
	alias = strings.ToLower(alias)
	key = strings.ToLower(key)
	if alias == key {
		return
	}
	a._appConfigDoWithWLock(func() {
		if a.aliasLookup(key) == alias {
			return
		}
		a.vp.RegisterAlias(alias, key)
		a.aliases[alias] = key
	})
}
*/

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
		a.delFastBoolCache(prop)
		a.vp.Set(prop, val)
	})
}

// Set default value for the prop
func (a *AppConfig) SetDefProp(prop string, defVal any) {
	doWithWriteLock(a, func() {
		a.delFastBoolCache(prop)
		a.vp.SetDefault(prop, defVal)
	})
}

func (a *AppConfig) delFastBoolCache(prop string) {
	prop = a.aliasLookup(strings.ToLower(prop))
	a.fastBoolCache.Del(prop)
}

// Check whether the prop exists
func (a *AppConfig) HasProp(prop string) bool {
	return returnWithReadLock(a, func() bool { return a.vp.IsSet(prop) })
}

// Get prop as int slice
func (a *AppConfig) GetPropIntSlice(prop string) []int {
	return returnWithReadLock(a, func() []int { return a.vp.GetIntSlice(prop) })
}

// Get prop immediate child names
func GetPropChild(prop string) []string {
	m := GetPropAny(prop)
	if m == nil {
		return []string{}
	}
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		return []string{}
	}
	mk := rv.MapKeys()
	c := []string{}
	for _, k := range mk {
		if k.Kind() == reflect.String {
			c = append(c, k.String())
		}
	}
	return c
}

// Get prop as string slice
func (a *AppConfig) GetPropStrSlice(prop string) []string {
	return returnWithReadLock(a, func() []string {
		v := a.vp.Get(prop)
		if s, ok := v.(string); ok {
			return strutil.SplitStr(s, ",")
		}
		return cast.ToStringSlice(v)
	})
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
//
// Deprecated: Use GetPropDuration() instead.
func (a *AppConfig) GetPropDur(prop string, unit time.Duration) time.Duration {
	return time.Duration(a.GetPropInt(prop)) * unit
}

// Get prop as time.Duration
func (a *AppConfig) GetPropDuration(prop string) time.Duration {
	return cast.ToDuration(a.GetPropStr(prop))
}

// Get prop as any
func (a *AppConfig) GetPropAny(prop string) any {
	return returnWithReadLock(a, func() any {
		return a.vp.Get(prop)
	})
}

// Get prop as bool
func (a *AppConfig) GetPropBool(prop string) bool {
	return returnWithReadLock(a, func() bool {
		prop = a.aliasLookup(strings.ToLower(prop))
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

// Same as GetPropStr() except the returned string is trimmed
func (a *AppConfig) GetPropStrTrimmed(prop string) string {
	v := a.GetPropStr(prop)
	if v == "" {
		return v
	}
	return strings.TrimSpace(v)
}

func (a *AppConfig) unmarshalMatchName(mapKey, fieldName string) bool {
	return strings.EqualFold(strings.ReplaceAll(mapKey, "-", ""), fieldName)
}

// Unmarshal configuration.
func (a *AppConfig) UnmarshalFromProp(ptr any) {
	doWithReadLock(a, func() {
		if err := a.vp.Unmarshal(ptr, func(dc *mapstructure.DecoderConfig) {
			dc.MatchName = a.unmarshalMatchName
		}); err != nil {
			Warnf("failed to UnmarshalFromProp, %v", err)
		}
	})
}

// Unmarshal configuration from a speicific key.
func (a *AppConfig) UnmarshalFromPropKey(key string, ptr any) {
	doWithReadLock(a, func() {
		if err := a.vp.UnmarshalKey(key, ptr, func(dc *mapstructure.DecoderConfig) {
			dc.MatchName = a.unmarshalMatchName
		}); err != nil {
			Warnf("failed to UnmarshalFromPropKey, %v", err)
		}
	})
}

// Overwrite existing conf using environment and cli args.
func (a *AppConfig) OverwriteConf(args []string) {
	// overwrite loaded configuration with environment variables
	a.overwriteConf(buildArgKeyValMap(os.Environ(), true), "Environment Variables")
	// overwrite the loaded configuration with cli arguments
	a.overwriteConf(buildArgKeyValMap(args, false), "CLI Args")
}

// Default way to read config file.
//
// Normally, this func is called by *MisoApp. Use this only when it's necessary. and you should call this func only once.
//
// The loaded configuration can be overriden by the cli arguments and environment variables.
func (a *AppConfig) DefaultReadConfig(args []string) {
	loaded := hash.NewSet[string]()

	defConfigFile := GuessConfigFilePath(args)
	loaded.Add(defConfigFile)

	if err := a.LoadConfigFromFile(defConfigFile); err != nil {
		Debugf("Failed to load config file, file: %v, %v", defConfigFile, err)
	} else {
		Infof("Loaded config file: %v", defConfigFile)
		a.defaultConfigFileLoaded = append(a.defaultConfigFileLoaded, defConfigFile)
	}

	// the load config file may specifiy extra files to be loaded
	extraFiles := a.GetPropStrSlice(PropConfigExtraFiles)

	for i := range extraFiles {
		f := extraFiles[i]

		if !loaded.Add(f) {
			continue
		}

		if ok, err := osutil.FileExists(f); err != nil || !ok {
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
			a.defaultConfigFileLoaded = append(a.defaultConfigFileLoaded, f)
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
			a.defaultConfigFileLoaded = append(a.defaultConfigFileLoaded, f)
		}
	}
}

func (a *AppConfig) GetDefaultConfigFileLoaded() []string {
	return slutil.SliceCopy(a.defaultConfigFileLoaded)
}

// Load config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method overides previously loaded config.
func (a *AppConfig) LoadConfigFromReader(reader io.Reader) error {
	var eo error

	doWithWriteLock(a, func() {
		if err := a.vp.MergeConfig(reader); err != nil {
			eo = fmt.Errorf("failed to load config from reader: %w", err)
		}

		// reset the whole fastBoolCache
		a.fastBoolCache = hash.NewStrRWMap[bool]()
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

// Reload config from string.
//
// Calling this method completely reloads previously loaded config.
func (a *AppConfig) ReloadConfigFromStr(sl ...string) error {

	for _, c := range sl {
		if c != "" {
			// test yaml format before we load anything into viper
			//
			// if viper.ReadConfig() failed, all configs are lost, we have to avoid that.
			var tmp map[string]interface{}
			if err := yaml.Unmarshal(util.UnsafeStr2Byt(c), &tmp); err != nil {
				return errs.Wrapf(err, "Failed reload nacos configs, invalid format")
			}
		}
	}

	var eo error
	doWithWriteLock(a, func() {
		for i, s := range sl {
			sr := bytes.NewReader(util.UnsafeStr2Byt(s))
			if i == 0 {
				if err := a.vp.ReadConfig(sr); err != nil {
					eo = fmt.Errorf("failed to reload config: %w", err)
					return
				}
			} else {
				if err := a.vp.MergeConfig(sr); err != nil {
					eo = fmt.Errorf("failed to reload config: %w", err)
					return
				}
			}
		}

		// reset the whole fastBoolCache
		a.fastBoolCache = hash.NewStrRWMap[bool]()
	})
	return eo
}

// Reload config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method completely reloads previously loaded config.
func (a *AppConfig) ReloadConfigFromReader(reader io.Reader) error {
	var eo error

	doWithWriteLock(a, func() {
		if err := a.vp.ReadConfig(reader); err != nil {
			eo = fmt.Errorf("failed to reload config from reader: %w", err)
		}

		// reset the whole fastBoolCache
		a.fastBoolCache = hash.NewStrRWMap[bool]()
	})

	return eo
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

func (a *AppConfig) overwriteConf(kvs map[string][]string, src string) {
	for k, v := range kvs {
		var vv any = v
		if len(v) == 1 {
			vv = v[0]
		}
		prevSet := a.HasProp(k)
		if prevSet {
			if _, ok := a.GetPropAny(k).(map[string]interface{}); ok {
				// k is a parent node
				prevSet = false
			}
		}
		a.SetProp(k, vv)
		Infof("Overwrote config: '%v', source: %v", k, src)
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
		pair := strings.SplitN(key, ":", 2)
		key = pair[0]
		defVal := s
		if len(pair) > 1 {
			defVal = strings.TrimSpace(pair[1])
		} else {
			defVal = ""
		}
		val := GetEnv(key)
		if val == "" {
			val = a.GetPropStr(key)
		}
		if val == "" {
			val = defVal
		}
		return val
	})
}

func newAppConfig() *AppConfig {
	ac := &AppConfig{
		vp:            viper.New(),
		rwmu:          &sync.RWMutex{},
		fastBoolCache: hash.NewStrRWMap[bool](),
		// aliases:       map[string]string{},
	}
	ac.vp.SetConfigType("yml")
	return ac
}

// Register alias.
//
// Only use this for backward compatibility.
//
// E.g.,
//
//	miso.RegisterAlias(newKey, oldkey)
//
// It may not work as expected if you are trying to call GetPropAny() on root node of a subtree.
//
// It doesn't work when your are loading new keys from yaml content, because viper doesn't support alias well.
// Try your best not to use it.
/*
	func RegisterAlias(alias, key string) {
		globalConfig().RegisterAlias(alias, key)
	}
*/

// Set value for the prop
func SetProp(prop string, val any) {
	globalConfig().SetProp(prop, val)
}

// Set default value for the prop
func SetDefProp(prop string, defVal any) {
	App().Config().SetDefProp(prop, defVal)
}

// Check whether the prop exists
func HasProp(prop string) bool {
	return globalConfig().HasProp(prop)
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
//
// Deprecated: Use GetPropDuration() instead.
func GetPropDur(prop string, unit time.Duration) time.Duration {
	return globalConfig().GetPropDur(prop, unit)
}

// Get prop as time.Duration
func GetPropDuration(prop string) time.Duration {
	return globalConfig().GetPropDuration(prop)
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

// Get prop as any
func GetPropAny(prop string) any {
	return globalConfig().GetPropAny(prop)
}

// Same as GetPropStr() except the returned string is trimmed
func GetPropStrTrimmed(prop string) string {
	return globalConfig().GetPropStrTrimmed(prop)
}

// Unmarshal configuration.
func UnmarshalFromProp(ptr any) {
	globalConfig().UnmarshalFromProp(ptr)
}

// Unmarshal configuration.
func UnmarshalFromPropAs[T any](ptr any) T {
	var t T
	UnmarshalFromProp(&t)
	return t
}

// Unmarshal configuration from a speicific key.
func UnmarshalFromPropKey(key string, ptr any) {
	globalConfig().UnmarshalFromPropKey(key, ptr)
}

// Unmarshal configuration from a speicific key.
func UnmarshalFromPropKeyAs[T any](key string) T {
	var t T
	UnmarshalFromPropKey(key, &t)
	return t
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

// Reload config from string.
//
// Calling this method completely reloads previously loaded config.
func ReloadConfigFromStr(s ...string) error {
	return globalConfig().ReloadConfigFromStr(s...)
}

// Reload config from io Reader.
//
// It's the caller's responsibility to close the provided reader.
//
// Calling this method completely reloads previously loaded config.
func ReloadConfigFromReader(reader io.Reader) error {
	return globalConfig().ReloadConfigFromReader(reader)
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
	if strutil.IsBlankStr(address) || address == util.LocalIpAny {
		address = util.GetLocalIPV4()
	}
	return address
}

var argKeyValRegex = regexp.MustCompile("[_]+")

// Parse CLI args to key-value map
func ArgKeyVal(args []string) map[string][]string {
	m := map[string][]string{}
	doAppend := func(key, val string) {
		if prev, ok := m[key]; ok {
			m[key] = append(prev, val)
		} else {
			m[key] = []string{val}
		}
	}
	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(s[:eq]))
		val := strings.TrimSpace(s[eq+1:])
		doAppend(key, val)
	}
	return m
}

// Parse CLI args to key-value map
func buildArgKeyValMap(args []string, requirePrefix bool) map[string][]string {
	m := map[string][]string{}
	doAppend := func(key, val string) {
		if prev, ok := m[key]; ok {
			m[key] = append(prev, val)
		} else {
			m[key] = []string{val}
		}
	}
	for _, s := range args {
		var eq int = strings.Index(s, "=")
		if eq == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(s[:eq]))
		val := strings.TrimSpace(s[eq+1:])
		if !requirePrefix {
			doAppend(key, val)
			continue
		}

		// e.g., 'miso_nacos_server_address' becomes 'nacos.server.address'
		if key2, ok := strutil.CutPrefixIgnoreCase(key, "miso_"); ok {
			key2 = argKeyValRegex.ReplaceAllLiteralString(key2, ".")
			doAppend(key2, val)
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
//
// See [ChangeDefaultConfigFilename].
func GuessConfigFilePath(args []string) string {
	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" })
	if strings.TrimSpace(path) == "" {
		path = defConfigFilename
	}
	return path
}

/*
Parse CLI Arg to extract a value from arg, [key]=[value]

e.g.,

To look for 'configFile=?'.

	path := ExtractArgValue(args, func(key string) bool { return key == "configFile" }).
*/
func ExtractArgValue(args []string, predicate func(t string) bool) string {
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
	return App().Config()
}

// Change default config filename, by default it's conf.yml.
func ChangeDefaultConfigFilename(f string) {
	f = strings.TrimSpace(f)
	if f == "" {
		panic(errors.New("default config filename is empty"))
	}
	defConfigFilename = f
}
