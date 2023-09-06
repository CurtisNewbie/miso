package miso

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

var (
	// Global handle to the redis
	redisp = &redisHolder{client: nil}
)

type redisHolder struct {
	client *redis.Client
	mu     sync.RWMutex
}

func init() {
	SetDefProp(PROP_REDIS_ENABLED, false)
	SetDefProp(PROP_REDIS_ADDRESS, "localhost")
	SetDefProp(PROP_REDIS_PORT, 6379)
	SetDefProp(PROP_REDIS_USERNAME, "")
	SetDefProp(PROP_REDIS_PASSWORD, "")
	SetDefProp(PROP_REDIS_DATABASE, 0)
}

/*
	Check if redis is enabled

	This func looks for following prop:

		"redis.enabled"

*/
func IsRedisEnabled() bool {
	return GetPropBool(PROP_REDIS_ENABLED)
}

/*
	Get Redis client

	Must call InitRedis(...) method before this method.
*/
func GetRedis() *redis.Client {
	redisp.mu.RLock()
	defer redisp.mu.RUnlock()

	if redisp.client == nil {
		panic("Redis Connection hasn't been initialized yet")
	}
	return redisp.client
}

// Get String
func GetStr(key string) (string, error) {
	scmd := GetRedis().Get(key)
	e := scmd.Err()
	if e != nil {
		if errors.Is(e, redis.Nil) {
			return "", nil
		}
		return "", e
	}
	return scmd.Val(), nil
}

/*
	Initialize redis client from configuration

	If redis client has been initialized, current func call will be ignored.

	This func looks for following prop:

		"redis.address"
		"redis.port"
		"redis.username"
		"redis.password"
		"redis.database"
*/
func InitRedisFromProp() (*redis.Client, error) {
	return InitRedis(
		GetPropStr(PROP_REDIS_ADDRESS),
		GetPropStr(PROP_REDIS_PORT),
		GetPropStr(PROP_REDIS_USERNAME),
		GetPropStr(PROP_REDIS_PASSWORD),
		GetPropInt(PROP_REDIS_DATABASE))
}

/*
	Initialize redis client

	If redis client has been initialized, current func call will be ignored
*/
func InitRedis(address string, port string, username string, password string, db int) (*redis.Client, error) {
	if IsRedisClientInitialized() {
		return GetRedis(), nil
	}

	redisp.mu.Lock()
	defer redisp.mu.Unlock()

	if redisp.client != nil {
		return redisp.client, nil
	}

	logrus.Infof("Connecting to redis '%v:%v', database: %v", address, port, db)
	var rdb *redis.Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", address, port),
		Password: password,
		DB:       db,
	})

	cmd := rdb.Ping()
	if cmd.Err() != nil {
		return nil, TraceErrf(cmd.Err(), "ping redis failed")
	}

	logrus.Info("Redis connection initialized")
	redisp.client = rdb
	return rdb, nil
}

// Check whether redis client is initialized
func IsRedisClientInitialized() bool {
	redisp.mu.RLock()
	defer redisp.mu.RUnlock()
	return redisp.client != nil
}
