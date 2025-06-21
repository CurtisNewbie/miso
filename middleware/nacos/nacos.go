package nacos

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var module = miso.InitAppModuleFunc(func() *nacosModule {
	return &nacosModule{
		mut:            &sync.RWMutex{},
		configContent:  util.NewStrRWMap[string](),
		watchedConfigs: make([]watchingConfig, 0, 1),
		reloadMut:      &sync.Mutex{},
	}
})

var (
	completeReload = &atomic.Bool{}
)

func init() {
	completeReload.Store(true)
	miso.RegisterConfigLoader(func(rail miso.Rail) error {
		return NacosBootstrap(rail)
	})
}

// Bootstrap Nacos Config Center
//
// In most cases, this should be called by miso itself when server bootstraps.
func NacosBootstrap(rail miso.Rail) error {
	if !miso.GetPropBool(PropNacosEnabled) {
		miso.Debug("nacos disabled")
		return nil
	}

	ok, err := module().init(rail)
	if err != nil {
		return miso.WrapErrf(err, "failed to initialize nacos module")
	}
	if !ok {
		miso.Debug("nacos already initialized")
		return nil // already initialized
	}
	rail.Info("Nacos Config Client Bootstrapped")

	// deregister on shutdown, we specify the order explicitly to make sure the service
	// is deregistered before shutting down the web server
	miso.AddOrderedShutdownHook(miso.DefShutdownOrder-1, func() {
		rail := miso.EmptyRail()
		module().shutdown(rail)
	})

	return nil
}

type nacosModule struct {
	mut            *sync.RWMutex
	initialized    bool
	configClient   config_client.IConfigClient
	onConfigChange []func()
	configContent  *util.StrRWMap[string]
	preloadedFiles []string
	watchedConfigs []watchingConfig
	reloadMut      *sync.Mutex
}

func (m *nacosModule) init(rail miso.Rail) (bool, error) {
	m.mut.Lock()
	defer m.mut.Unlock()
	if m.initialized {
		return false, nil
	}

	// preserve content of the already loaded config files
	loadedConfigFiles := miso.App().Config().GetDefaultConfigFileLoaded()
	for _, f := range util.Distinct(loadedConfigFiles) {
		if f == "" {
			continue
		}
		contentByte, err := util.ReadFileAll(f)
		if err != nil {
			return false, miso.WrapErr(err)
		}
		content := strings.TrimSpace(string(contentByte))
		if content != "" {
			m.configContent.Put("file:"+f, content)

			if miso.IsTraceLevel() {
				rail.Tracef("Preserved the already loaded config file: %v\n%v", f, content)
			} else {
				rail.Debugf("Preserved the already loaded config file: %v", f)
			}
			m.preloadedFiles = append(m.preloadedFiles, f)
		}
	}

	clientConfig, serverConfigs, err := m.buildConfig(rail)
	if err != nil {
		return false, err
	}
	cc, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return false, miso.WrapErrf(err, "failed to create nacos config client")
	}
	m.configClient = cc

	mergeConfig := func(w watchingConfig) error {
		// fetch config on bootstrap
		p := vo.ConfigParam{DataId: w.DataId, Group: w.Group}
		configStr, err := m.configClient.GetConfig(p)
		if err != nil {
			return miso.WrapErrf(err, "failed to fetch nacos config, param: %#v", p)
		}
		if err := miso.LoadConfigFromStr(configStr, rail); err != nil {
			rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", w.Group, w.DataId, configStr)
		}
		rail.Tracef("Fetched nacos config, %v-%v:\n%v", w.Group, w.DataId, configStr)
		m.configContent.Put(w.Key(), configStr)
		return nil
	}

	appDataId := miso.GetPropStr(PropNacosConfigDataId)
	appGroup := miso.GetPropStr(PropNacosConfigGroup)
	if util.IsBlankStr(appDataId) {
		return false, miso.NewErrf("Missing configuration: '%v'", PropNacosConfigDataId)
	}
	appConfig := watchingConfig{DataId: appDataId, Group: appGroup}

	// merge app's nacos config first before we read `PropNacosConfigWatch`
	if err := mergeConfig(appConfig); err != nil {
		return false, err
	}

	watched := miso.GetPropStrSlice(PropNacosConfigWatch)
	if len(watched) > 0 {
		rail.Debugf("watched: %#v", watched)
	}

	for _, w := range watched {
		tok := strings.SplitN(w, ":", 2)
		if len(tok) > 0 {
			dataId := strings.TrimSpace(tok[0])
			if dataId == "" {
				continue
			}
			group := ""
			if len(tok) > 1 {
				group = strings.TrimSpace(tok[1])
			}
			if group == "" {
				group = "DEFAULT_GROUP"
			}
			w := watchingConfig{DataId: dataId, Group: group}
			m.watchedConfigs = append(m.watchedConfigs, w)
			if err := mergeConfig(w); err != nil {
				return false, err
			}
		}
	}

	// subscribe changes
	m.watchedConfigs = append(m.watchedConfigs, appConfig) // app's config
	for _, w := range m.watchedConfigs {
		rail.Infof("Listening nacos config: %#v", w)
		m.configClient.ListenConfig(vo.ConfigParam{
			DataId: w.DataId,
			Group:  w.Group,
			OnChange: func(namespace, group, dataId, data string) {
				rail := miso.EmptyRail()
				rail.Infof("nacos config changed, %v-%v", group, dataId)
				w := watchingConfig{DataId: dataId, Group: group}
				if completeReload.Load() {
					m.configContent.Put(w.Key(), data)
					m.reloadConfigs(rail)
				} else {
					rail.Tracef("Loading nacos config:\n%v", data)
					if err := miso.LoadConfigFromStr(data, rail); err != nil {
						rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", group, dataId, data)
					}
				}

				m.mut.RLock()
				defer m.mut.RUnlock()
				for _, cbk := range m.onConfigChange {
					go cbk()
				}
			},
		})
	}

	m.initialized = true
	return true, nil
}

func (m *nacosModule) shutdown(rail miso.Rail) {

}

func (m *nacosModule) reloadConfigs(rail miso.Rail) {
	m.reloadMut.Lock()
	defer m.reloadMut.Unlock()

	wcl := make([]string, 0, len(m.preloadedFiles)+len(m.watchedConfigs))
	for _, f := range m.preloadedFiles {
		if c, ok := m.configContent.Get("file:" + f); ok {
			c = strings.TrimSpace(c)
			if miso.IsTraceLevel() {
				rail.Tracef("Reloading preloaded config file, %v:\n%v", f, c)
			} else {
				rail.Debugf("Reloading preloaded config file, %v", f)
			}
			wcl = append(wcl, c)
		}
	}

	for _, w := range m.watchedConfigs {
		if c, ok := m.configContent.Get(w.Key()); ok {
			c = strings.TrimSpace(c)
			if miso.IsTraceLevel() {
				rail.Tracef("Reloading nacos config, %v-%v:\n%v", w.Group, w.DataId, c)
			} else {
				rail.Debugf("Reloading nacos config, %v-%v", w.Group, w.DataId)
			}
			wcl = append(wcl, c)
		}
	}
	if err := miso.ReloadConfigFromStr(wcl...); err != nil {
		rail.Errorf("Failed reload nacos configs, %v", err)
	}
}

func (m *nacosModule) buildConfig(rail miso.Rail) (constant.ClientConfig, []constant.ServerConfig, error) {
	ns := miso.GetPropStr(PropNacosServerNamespace)
	un := miso.GetPropStr(PropNacosServerUsername)
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(ns),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithCacheDir(miso.GetPropStr(PropNacosCacheDir)),
		constant.WithUsername(un),
		constant.WithPassword(miso.GetPropStr(PropNacosServerPassword)),
		constant.WithCustomLogger(&nacosLogger{}),
	)

	port := miso.GetPropInt(PropNacosServerPort)
	serverAddr := strings.TrimSpace(miso.GetPropStr(PropNacosServerAddr))
	if serverAddr == "" {
		return constant.ClientConfig{}, nil, miso.NewErrf("Missing config: '%v'", PropNacosServerAddr)
	}
	scheme := miso.GetPropStr(PropNacosServerScheme)
	if s, ok := util.CutPrefixIgnoreCase(serverAddr, "http://"); ok {
		scheme = "http"
		serverAddr = s
	} else if s, ok := util.CutPrefixIgnoreCase(serverAddr, "https://"); ok {
		scheme = "https"
		serverAddr = s
		if port == 0 {
			port = 443
		}
	}
	if port == 0 {
		port = 80
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      serverAddr,
			ContextPath: miso.GetPropStr(PropNacosServerContextPath),
			Scheme:      scheme,
			Port:        uint64(port),
		},
	}
	rail.Infof("Connecting to Nacos Server: %v, ns: %v, user: %v", serverAddr, ns, un)
	return clientConfig, serverConfigs, nil
}

func OnConfigChanged(f func()) {
	m := module()
	m.mut.Lock()
	defer m.mut.Unlock()
	m.onConfigChange = append(m.onConfigChange, f)
}

// Whether we should completely reload existing configs with nacos configs, by default it's true.
//
// This is usually used when all the configurations are managed on nacos.
//
// If a key xxx is removed from nacos, then this key is unset as well, because the config map is recreated.
// However, overrides and defaults will still exist, e.g., SetProp(), SetDefProp().
func ReloadConfigsOnChange(v bool) {
	completeReload.Store(v)
}

type nacosLogger struct {
}

func (n *nacosLogger) Info(args ...interface{})               {}
func (n *nacosLogger) Warn(args ...interface{})               { miso.Warn(args...) }
func (n *nacosLogger) Error(args ...interface{})              { miso.Error(args...) }
func (n *nacosLogger) Debug(args ...interface{})              {}
func (n *nacosLogger) Infof(fmt string, args ...interface{})  {}
func (n *nacosLogger) Warnf(fmt string, args ...interface{})  { miso.Warnf(fmt, args...) }
func (n *nacosLogger) Errorf(fmt string, args ...interface{}) { miso.Errorf(fmt, args...) }
func (n *nacosLogger) Debugf(fmt string, args ...interface{}) {}

type watchingConfig struct {
	DataId string
	Group  string
}

func (w watchingConfig) Key() string {
	return w.DataId + ":" + w.Group
}
