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

	// misoconfig-prop: nacos server port (by default it's either 80 or 443)
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
	miso.SetDefProp(PropNacosCacheDir, "/tmp/nacos/cache")
}

// misoconfig-default-end
