package redis

import (
	"fmt"
	"log"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/go-redis/redis"
)

var (

	// Global handle to the redis
	redisHandle *redis.Client
)

// Get Redis Handle, must call InitRedis(...) method before this method
func GetRedis() *redis.Client {
	if redisHandle == nil {
		panic("GetRedis is called prior to the Redis Handle initialization, this is illegal, see InitRedis(...) method")
	}
	return redisHandle
}

// Init handle to redis
func InitRedisFromConfig(config *config.RedisConfig) *redis.Client {
	return InitRedis(config.Address, config.Port, config.Username, config.Password, config.Database)
}

// Init handle to redis
func InitRedis(address string, port string, username string, password string, db int) *redis.Client {
	log.Printf("Connecting to redis '%v:%v', database: %v", address, port, db)
	var rdb *redis.Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", address, port),
		Password: password,
		DB:       db,
	})

	redisHandle = rdb
	log.Println("Redis Handle initialized")

	return rdb
}
