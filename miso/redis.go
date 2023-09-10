package miso

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-redis/redis"
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
func InitRedisFromProp(rail Rail) (*redis.Client, error) {
	return InitRedis(
		rail,
		RedisConnParam{
			Address:  GetPropStr(PROP_REDIS_ADDRESS),
			Port:     GetPropStr(PROP_REDIS_PORT),
			Username: GetPropStr(PROP_REDIS_USERNAME),
			Password: GetPropStr(PROP_REDIS_PASSWORD),
			Db:       GetPropInt(PROP_REDIS_DATABASE),
		})
}

type RedisConnParam struct {
	Address  string
	Port     string
	Username string
	Password string
	Db       int
}

/*
Initialize redis client

If redis client has been initialized, current func call will be ignored
*/
func InitRedis(rail Rail, p RedisConnParam) (*redis.Client, error) {
	if IsRedisClientInitialized() {
		return GetRedis(), nil
	}

	redisp.mu.Lock()
	defer redisp.mu.Unlock()

	if redisp.client != nil {
		return redisp.client, nil
	}

	rail.Infof("Connecting to redis '%v:%v', database: %v", p.Address, p.Port, p.Db)
	var rdb *redis.Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", p.Address, p.Port),
		Password: p.Password,
		DB:       p.Db,
	})

	cmd := rdb.Ping()
	if cmd.Err() != nil {
		return nil, TraceErrf(cmd.Err(), "ping redis failed")
	}

	rail.Info("Redis connection initialized")
	redisp.client = rdb
	return rdb, nil
}

// Check whether redis client is initialized
func IsRedisClientInitialized() bool {
	redisp.mu.RLock()
	defer redisp.mu.RUnlock()
	return redisp.client != nil
}
