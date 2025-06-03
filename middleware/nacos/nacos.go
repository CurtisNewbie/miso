package nacos

import (
	"sync"

	"github.com/curtisnewbie/miso/miso"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var module = miso.InitAppModuleFunc(func() *nacosModule {
	return &nacosModule{
		mut: &sync.RWMutex{},
	}
})

func init() {
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Boostrap Nacos Config Center",
		Bootstrap: nacosBootstrap,
		Condition: nacosBootstrapCondition,
		Order:     miso.BootstrapOrderL4,
	})
}

func nacosBootstrap(rail miso.Rail) error {

	if err := module().init(rail); err != nil {
		return miso.WrapErrf(err, "Failed to initialize nacos module")
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

func nacosBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropNacosEnabled), nil
}

type nacosModule struct {
	mut            *sync.RWMutex
	configClient   config_client.IConfigClient
	onConfigChange []func()
}

func (m *nacosModule) init(rail miso.Rail) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	clientConfig, serverConfigs := m.buildConfig()
	cc, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return miso.WrapErrf(err, "failed to create nacos config client")
	}
	m.configClient = cc

	dataId := miso.GetPropStr(PropNacosConfigDataId)
	group := miso.GetPropStr(PropNacosConfigGroup)
	if dataId != "" {

		// fetch config on bootstrap
		configStr, err := m.configClient.GetConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  group,
		})
		if err != nil {
			return err
		}
		if err := miso.LoadConfigFromStr(configStr, rail); err != nil {
			rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", group, dataId, configStr)
		}

		// subscribe changes
		rail.Infof("Listening nacos config: %v-%v", group, dataId)
		m.configClient.ListenConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  group,
			OnChange: func(namespace, group, dataId, data string) {
				rail := miso.EmptyRail()
				rail.Infof("nacos config changed, %v-%v", group, dataId)
				if err := miso.LoadConfigFromStr(data, rail); err != nil {
					rail.Errorf("Failed to merge Nacos config, %v-%v\n%v", group, dataId, data)
				}

				m.mut.RLock()
				defer m.mut.RUnlock()
				for _, cbk := range m.onConfigChange {
					go cbk()
				}
			},
		})
	}

	return nil
}

func (m *nacosModule) shutdown(rail miso.Rail) {
}

func (m *nacosModule) buildConfig() (constant.ClientConfig, []constant.ServerConfig) {
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(miso.GetPropStr(PropNacosServerNamespace)),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithCacheDir(miso.GetPropStr(PropNacosCacheDir)),
		constant.WithUsername(miso.GetPropStr(PropNacosServerUsername)),
		constant.WithPassword(miso.GetPropStr(PropNacosServerPassword)),
		constant.WithCustomLogger(&nacosLogger{}),
	)

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      miso.GetPropStr(PropNacosServerAddr),
			ContextPath: miso.GetPropStr(PropNacosServerContextPath),
			Scheme:      miso.GetPropStr(PropNacosServerScheme),
			Port:        uint64(miso.GetPropInt(PropNacosServerPort)),
		},
	}
	return clientConfig, serverConfigs
}

func OnConfigChanged(f func()) {
	m := module()
	m.mut.Lock()
	defer m.mut.Unlock()
	m.onConfigChange = append(m.onConfigChange, f)
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
