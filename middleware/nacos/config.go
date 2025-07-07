package nacos

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Nacos Configuration
const (

	// misoconfig-prop: enable nacos client | false
	PropNacosEnabled = "nacos.enabled"

	// misoconfig-prop: nacos server address | localhost
	PropNacosServerAddr = "nacos.server.addr"

	// misoconfig-prop: nacos server address scheme | http
	PropNacosServerScheme = "nacos.server.scheme"

	// misoconfig-prop: nacos server port (by default it's either 80, 443 or 8848)
	PropNacosServerPort = "nacos.server.port"

	// misoconfig-prop: nacos server context path |
	PropNacosServerContextPath = "nacos.server.context-path"

	// misoconfig-prop: nacos server namespace |
	PropNacosServerNamespace = "nacos.server.namespace"

	// misoconfig-prop: nacos server username |
	PropNacosServerUsername = "nacos.server.username"

	// misoconfig-prop: nacos server password |
	PropNacosServerPassword = "nacos.server.password"

	// misoconfig-prop: nacos config data-id | ${app.name}
	PropNacosConfigDataId = "nacos.server.config.data-id"

	// misoconfig-prop: nacos config group | DEFAULT_GROUP
	PropNacosConfigGroup = "nacos.server.config.group"

	// misoconfig-prop: extra watched nacos config, (slice of strings, format: `"${data-id}" + ":" + "${group}"`)
	PropNacosConfigWatch = "nacos.server.config.watch"

	// misoconfig-prop: enable nacos client for service discovery | true
	PropNacosDiscoveryEnabled = "nacos.discovery.enabled"

	// misoconfig-prop: register current instance on nacos for service discovery | true
	PropNacosDiscoveryRegisterInstance = "nacos.discovery.register-instance"

	// misoconfig-prop: register service address | `"${server.host}"`
	PropNacosDiscoveryRegisterAddress = "nacos.discovery.register-address"

	// misoconfig-prop: register service name | `"${app.name}"`
	PropNacosDiscoveryRegisterName = "nacos.discovery.register-name"

	// misoconfig-prop: enable endpoint for manual Nacos service deregistration | false
	PropNacosDiscoveryEnableDeregisterUrl = "nacos.discovery.enable-deregister-url"

	// misoconfig-prop: endpoint url for manual Nacos service deregistration | /nacos/deregister
	PropNacosDiscoveryDeregisterUrl = "nacos.discovery.deregister-url"

	// misoconfig-prop: instance metadata (`map[string]string`)
	PropNacosDiscoveryMetadata = "nacos.discovery.metadata"

	// misoconfig-prop: nacos cache dir | /tmp/nacos/cache
	PropNacosCacheDir = "nacos.cache-dir"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropNacosEnabled, false)
	miso.SetDefProp(PropNacosServerAddr, "localhost")
	miso.SetDefProp(PropNacosServerScheme, "http")
	miso.SetDefProp(PropNacosConfigDataId, "${app.name}")
	miso.SetDefProp(PropNacosConfigGroup, "DEFAULT_GROUP")
	miso.SetDefProp(PropNacosDiscoveryEnabled, false)
	miso.SetDefProp(PropNacosDiscoveryRegisterInstance, true)
	miso.SetDefProp(PropNacosDiscoveryRegisterAddress, "${server.host}")
	miso.SetDefProp(PropNacosDiscoveryRegisterName, "${app.name}")
	miso.SetDefProp(PropNacosDiscoveryEnableDeregisterUrl, false)
	miso.SetDefProp(PropNacosDiscoveryDeregisterUrl, "/nacos/deregister")
	miso.SetDefProp(PropNacosCacheDir, "/tmp/nacos/cache")
}

// misoconfig-default-end
