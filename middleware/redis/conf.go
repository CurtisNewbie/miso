package redis

import "github.com/curtisnewbie/miso/miso"

const (
	/*
		------------------------------------

		Prop for Redis

		------------------------------------
	*/
	PropRedisEnabled  = "redis.enabled"
	PropRedisAddress  = "redis.address"
	PropRedisPort     = "redis.port"
	PropRedisUsername = "redis.username"
	PropRedisPassword = "redis.password"
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
