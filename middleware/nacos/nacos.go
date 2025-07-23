package nacos

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/cast"
)

var module = miso.InitAppModuleFunc(func() *nacosModule {
	return &nacosModule{
		mut:            &sync.RWMutex{},
		configContent:  util.NewStrRWMap[string](),
		watchedConfigs: make([]watchingConfig, 0, 1),
		reloadMut:      &sync.Mutex{},
		serverList: &NacosServerList{
			watchedServices: util.NewSetPtr[string](),
			wsmu:            &sync.RWMutex{},
		},
	}
})

var (
	completeReload = &atomic.Bool{}

	_ miso.ServerList = (*NacosServerList)(nil)
)

func init() {
	completeReload.Store(true)
	miso.RegisterConfigLoader(func(rail miso.Rail) error {
		err := BootstrapConfigCenter(rail)
		if err != nil {
			return err
		}
		module().prepareDeregisterUrl(rail)
		return nil
	})

	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Boostrap Nacos Service Discovery",
		Bootstrap: BootstrapServiceDiscovery,
		Condition: func(rail miso.Rail) (bool, error) {
			return miso.GetPropBool(PropNacosEnabled) && miso.GetPropBool(PropNacosDiscoveryEnabled), nil
		},
		Order: miso.BootstrapOrderL4,
	})
}

// Bootstrap Nacos Config Center
//
// In most cases, this should be called by miso itself when server bootstraps.
func BootstrapConfigCenter(rail miso.Rail) error {
	if !miso.GetPropBool(PropNacosEnabled) {
		return nil
	}

	ok, err := module().initConfigCenter(rail)
	if err != nil {
		return miso.WrapErrf(err, "failed to initialize nacos module for config center")
	}
	if !ok {
		miso.Debug("nacos already initialized")
		return nil // already initialized
	}
	rail.Info("Nacos Config Client Bootstrapped")
	return nil
}

// Bootstrap Nacos ServiceDiscovery
//
// In most cases, this should be called by miso itself when server bootstraps.
func BootstrapServiceDiscovery(rail miso.Rail) error {

	ok, err := module().initDiscovery(rail)
	if err != nil {
		return miso.WrapErrf(err, "failed to initialize nacos module for service discovery")
	}
	if !ok {
		miso.Debug("nacos already initialized")
		return nil // already initialized
	}
	rail.Info("Nacos Service Discovery Bootstrapped")
	return nil
}

type nacosModule struct {
	mut                  *sync.RWMutex
	configInitialized    bool
	discoveryInitialized bool
	configClient         config_client.IConfigClient
	onConfigChange       []func()
	configContent        *util.StrRWMap[string]
	preloadedFiles       []string
	watchedConfigs       []watchingConfig
	reloadMut            *sync.Mutex
	serverList           *NacosServerList
}

func (m *nacosModule) prepareDeregisterUrl(rail miso.Rail) {
	if miso.GetPropBool(PropNacosDiscoveryEnabled) && miso.GetPropBool(PropNacosDiscoveryEnableDeregisterUrl) {
		deregisterUrl := miso.GetPropStr(PropNacosDiscoveryDeregisterUrl)
		if !util.IsBlankStr(deregisterUrl) {
			rail.Infof("Enabled 'GET %v' for manual nacos service deregistration", deregisterUrl)

			miso.HttpGet(deregisterUrl, miso.ResHandler(
				func(inb *miso.Inbound) (any, error) {
					_, r := inb.Unwrap()
					rail.Infof("Deregistering nacos service registration, remote_addr: %v", r.RemoteAddr)
					if err := deregisterNacosService(m.serverList.client); err != nil {
						rail.Errorf("failed to deregister nacos service, %v", err)
						return nil, err
					} else {
						rail.Info("Nacos service deregistered")
					}
					return nil, nil
				})).
				Desc("Endpoint used to trigger Nacos service deregistration")
		}
	}
}

func (m *nacosModule) initDiscovery(rail miso.Rail) (bool, error) {
	m.mut.Lock()
	defer m.mut.Unlock()
	if m.discoveryInitialized {
		return false, nil
	}

	clientConfig, serverConfigs, err := m.buildConfig(rail)
	if err != nil {
		return false, err
	}

	// setup api
	miso.ChangeGetServerList(func() miso.ServerList { return m.serverList })
	rail.Debug("Using Nacos based GetServerList")

	nc, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return false, miso.WrapErrf(err, "failed to create nacos naming client")
	}
	m.serverList.client = nc
	rail.Infof("Created nacos naming client")

	// deregister on shutdown, we specify the order explicitly to make sure the service
	// is deregistered before shutting down the web server
	miso.AddOrderedShutdownHook(miso.DefShutdownOrder-1, func() {
		rail := miso.EmptyRail()
		rail.Infof("Deregistering Nacos service")
		if e := deregisterNacosService(m.serverList.client); e != nil {
			rail.Errorf("Failed to deregister on Nacos, %v", e)
		}
	})

	// register current instance
	miso.OnAppReady(func(rail miso.Rail) error {
		if err := registerNacosService(nc); err != nil {
			return miso.WrapErrf(err, "failed to register on nacos")
		}
		return nil
	})

	m.discoveryInitialized = true
	return true, nil
}

func (m *nacosModule) initConfigCenter(rail miso.Rail) (bool, error) {
	m.mut.Lock()
	defer m.mut.Unlock()
	if m.configInitialized {
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

	mergeConfig := func(w watchingConfig) (string, error) {
		p := vo.ConfigParam{DataId: w.DataId, Group: w.Group}
		configStr, err := m.configClient.GetConfig(p)
		if err != nil {
			return "", miso.WrapErrf(err, "failed to fetch nacos config, param: %#v", p)
		}
		if err := miso.LoadConfigFromStr(configStr, rail); err != nil {
			rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", w.Group, w.DataId, configStr)
		}
		rail.Tracef("Fetched nacos config, %v-%v:\n%v", w.Group, w.DataId, configStr)
		m.configContent.Put(w.Key(), configStr)
		return configStr, nil
	}

	watchedKeys := util.NewSet[string]()
	addWatchConfig := func(w string) error {
		rail.Debugf("Parsing nacos watch config value: %v", w)
		tok := strings.SplitN(w, ":", 2)
		if len(tok) > 0 {
			dataId := strings.TrimSpace(tok[0])
			if dataId == "" {
				return nil
			}
			group := ""
			if len(tok) > 1 {
				group = strings.TrimSpace(tok[1])
			}
			if group == "" {
				group = "DEFAULT_GROUP"
			}
			w := watchingConfig{DataId: dataId, Group: group}
			if !watchedKeys.Add(w.Key()) {
				return nil
			}
			m.watchedConfigs = append(m.watchedConfigs, w)
			if _, err := mergeConfig(w); err != nil {
				return err
			}
		}
		return nil
	}
	loadWatchConfigsFromProp := func() error {
		v := miso.GetPropStrSlice(PropNacosConfigWatch)
		if len(v) < 1 {
			return nil
		}
		rail.Debug("Loading NacosConfigWatch Prop")
		for _, w := range v {
			if err := addWatchConfig(w); err != nil {
				return err
			}
		}
		return nil
	}

	// load watched configs before app's config
	if err := loadWatchConfigsFromProp(); err != nil {
		return false, err
	}

	// merge app's nacos config
	appDataId := miso.GetPropStr(PropNacosConfigDataId)
	appGroup := miso.GetPropStr(PropNacosConfigGroup)
	if util.IsBlankStr(appDataId) {
		return false, miso.NewErrf("Missing configuration: '%v'", PropNacosConfigDataId)
	}
	appConfig := watchingConfig{DataId: appDataId, Group: appGroup}
	appConfigStr, err := mergeConfig(appConfig)
	if err != nil {
		return false, err
	}

	// load watched configs after app's config
	if err := loadWatchConfigsFromProp(); err != nil {
		return false, err
	}

	// place app's configs on the top
	if err := miso.LoadConfigFromStr(appConfigStr, rail); err != nil {
		rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", appConfig.Group, appConfig.DataId, appConfigStr)
	}

	// subscribe changes
	// app's config is always placed at the end (it can override other configs)
	m.watchedConfigs = append(m.watchedConfigs, appConfig)
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

	OnConfigChanged(func() {
		miso.SetLogLevel(miso.GetPropStr(miso.PropLoggingLevel))
	})

	m.configInitialized = true
	return true, nil
}

func (m *nacosModule) reloadConfigs(rail miso.Rail) {
	start := time.Now()
	defer miso.TimeOp(rail, start, "nacos configs reload")

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
		constant.WithUpdateCacheWhenEmpty(true),
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

	contextPath := miso.GetPropStr(PropNacosServerContextPath)
	serverConfigs := []constant.ServerConfig{}
	scsb := []string{}
	for _, host := range strings.Split(serverAddr, ",") {
		if host == "" {
			continue
		}
		scheme := miso.GetPropStr(PropNacosServerScheme)
		if s, ok := util.CutPrefixIgnoreCase(host, "http://"); ok {
			scheme = "http"
			host = s
		} else if s, ok := util.CutPrefixIgnoreCase(host, "https://"); ok {
			scheme = "https"
			host = s
			if port == 0 {
				port = 443
			}
		}
		if port == 0 {
			port = 8848
		}
		host = strings.TrimSpace(host)
		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr:      host,
			ContextPath: contextPath,
			Scheme:      scheme,
			Port:        uint64(port),
		})
		scsb = append(scsb, fmt.Sprintf("%v:%v (%v)", host, port, scheme))
	}
	rail.Infof("Connecting to Nacos Server: %v, ns: %v, user: %v", strings.Join(scsb, ", "), ns, un)
	return clientConfig, serverConfigs, nil
}

func OnConfigChanged(f func()) {
	m := module()
	m.mut.Lock()
	defer m.mut.Unlock()
	m.onConfigChange = append(m.onConfigChange, f)
}

func LogConfigChanges(props ...string) {
	onChanged := func() {
		for _, p := range props {
			miso.Infof("Prop '%v': %v", p, miso.GetPropAny(p))
		}
	}
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		onChanged()
		return nil
	})
	OnConfigChanged(func() {
		onChanged()
	})
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

// Holder of a list of ServiceHolder
type NacosServerList struct {
	client          naming_client.INamingClient
	watchedServices *util.Set[string]
	wsmu            *sync.RWMutex
}

func (s *NacosServerList) ListServers(rail miso.Rail, name string) []miso.Server {
	s.wsmu.RLock()
	if !s.watchedServices.Has(name) {
		s.wsmu.RUnlock()

		s.wsmu.Lock()
		if s.watchedServices.Add(name) {
			err := s.client.Subscribe(&vo.SubscribeParam{
				ServiceName: name,
				SubscribeCallback: func(services []model.SubscribeService, err error) {
					rail.Infof("Service '%v' instances changed: %#v", name, services)
				},
			})
			if err != nil {
				rail.Errorf("Failed to subscribe service '%v', %v", name, err)
				s.watchedServices.Del(name)
			}
		}
		s.wsmu.Unlock()
	} else {
		s.wsmu.RUnlock()
	}

	inst, err := s.client.SelectAllInstances(vo.SelectAllInstancesParam{ServiceName: name})
	if err != nil {
		rail.Errorf("Failed to select instances for %v, %v", name, err)
		return nil
	}
	rail.Debugf("ListServers: %v, instances: %#v", name, inst)
	inst = util.CopyFilter(inst, func(i model.Instance) bool { return i.Enable && i.Weight > 0 && i.Healthy })
	return util.MapTo(inst, func(v model.Instance) miso.Server {
		return miso.Server{
			Address: v.Ip,
			Port:    int(v.Port),
			Meta:    util.MapCopy(v.Metadata),
		}
	})
}

func (s *NacosServerList) IsSubscribed(rail miso.Rail, service string) bool {
	return true
}

func (s *NacosServerList) Subscribe(rail miso.Rail, service string) error {
	return nil
}

func (s *NacosServerList) Unsubscribe(rail miso.Rail, service string) error {
	return nil
}

func (s *NacosServerList) PollInstance(rail miso.Rail, name string) error {
	return nil
}

func registerNacosService(nc naming_client.INamingClient) error {
	if !miso.GetPropBool(PropNacosDiscoveryRegisterInstance) {
		return nil
	}

	serverPort := miso.GetPropInt(miso.PropServerActualPort)
	registerName := miso.GetPropStr(PropNacosDiscoveryRegisterName)
	registerAddress := miso.GetPropStr(PropNacosDiscoveryRegisterAddress)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = miso.ResolveServerHost(miso.GetPropStr(miso.PropServerHost))
	} else {
		registerAddress = miso.ResolveServerHost(registerAddress)
	}

	meta := miso.GetPropStrMap(PropNacosDiscoveryMetadata)
	if meta == nil {
		meta = map[string]string{}
	}
	meta[miso.ServiceMetaRegisterTime] = cast.ToString(util.Now().UnixMilli())

	ok, err := nc.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: registerName,
		Ip:          registerAddress,
		Port:        uint64(serverPort),
		Weight:      1,
		Healthy:     true,
		Metadata:    meta,
		Ephemeral:   true,
		Enable:      true,
	})
	if err != nil {
		return miso.WrapErr(err)
	}
	if !ok {
		return miso.NewErrf("Register nacos service failed")
	}

	miso.Infof("Registered on nacos, %v %v:%v", registerName, registerAddress, serverPort)
	return nil
}

func deregisterNacosService(nc naming_client.INamingClient) error {
	if !miso.GetPropBool(PropNacosDiscoveryRegisterInstance) {
		return nil
	}

	serverPort := miso.GetPropInt(miso.PropServerActualPort)
	registerName := miso.GetPropStr(PropNacosDiscoveryRegisterName)
	registerAddress := miso.GetPropStr(PropNacosDiscoveryRegisterAddress)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = miso.ResolveServerHost(miso.GetPropStr(miso.PropServerHost))
	} else {
		registerAddress = miso.ResolveServerHost(registerAddress)
	}

	meta := miso.GetPropStrMap(PropNacosDiscoveryMetadata)
	if meta == nil {
		meta = map[string]string{}
	}
	meta[miso.ServiceMetaRegisterTime] = cast.ToString(util.Now().UnixMilli())

	_, err := nc.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: registerName,
		Ip:          registerAddress,
		Port:        uint64(serverPort),
		Ephemeral:   true,
	})
	if err != nil {
		return err
	}

	miso.Infof("Deregistered on nacos, %v %v:%v", registerName, registerAddress, serverPort)
	return nil
}

func DeregisterNacosService(rail miso.Rail) error {
	m := module()
	if err := deregisterNacosService(m.serverList.client); err != nil {
		rail.Errorf("failed to deregister nacos service, %v", err)
		return err
	} else {
		rail.Info("Nacos service deregistered")
		return nil
	}
}
