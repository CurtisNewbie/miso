package redis

import "github.com/curtisnewbie/miso/miso"

// misoapi-config-section: Redis Configuration
const (

	// misoapi-config: enable Redis client | false
	PropRedisEnabled = "redis.enabled"

	// misoapi-config: Redis server host | `localhost`
	PropRedisAddress = "redis.address"

	// misoapi-config: Redis server port | 6379
	PropRedisPort = "redis.port"

	// misoapi-config: username
	PropRedisUsername = "redis.username"

	// misoapi-config: password
	PropRedisPassword = "redis.password"

	// misoapi-config: database | 0
	PropRedisDatabase = "redis.database"
)

func init() {
	miso.SetDefProp(PropRedisEnabled, false)
	miso.SetDefProp(PropRedisAddress, "localhost")
	miso.SetDefProp(PropRedisPort, 6379)
	miso.SetDefProp(PropRedisUsername, "")
	miso.SetDefProp(PropRedisPassword, "")
	miso.SetDefProp(PropRedisDatabase, 0)
}
