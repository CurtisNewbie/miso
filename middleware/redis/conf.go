package redis

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Redis Configuration
const (

	// misoconfig-prop: enable Redis client | false
	PropRedisEnabled = "redis.enabled"

	// misoconfig-prop: Redis server host | `localhost`
	PropRedisAddress = "redis.address"

	// misoconfig-prop: Redis server port | 6379
	PropRedisPort = "redis.port"

	// misoconfig-prop: username
	PropRedisUsername = "redis.username"

	// misoconfig-prop: password
	PropRedisPassword = "redis.password"

	// misoconfig-prop: database | 0
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
