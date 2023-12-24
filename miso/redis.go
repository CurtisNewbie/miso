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
	SetDefProp(PropRedisEnabled, false)
	SetDefProp(PropRedisAddress, "localhost")
	SetDefProp(PropRedisPort, 6379)
	SetDefProp(PropRedisUsername, "")
	SetDefProp(PropRedisPassword, "")
	SetDefProp(PropRedisDatabas, 0)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Redis",
		Bootstrap: RedisBootstrap,
		Condition: RedisBootstrapCondition,
		Order:     -10,
	})
}

/*
Check if redis is enabled

This func looks for following prop:

	"redis.enabled"
*/
func IsRedisEnabled() bool {
	return GetPropBool(PropRedisEnabled)
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
			Address:  GetPropStr(PropRedisAddress),
			Port:     GetPropStr(PropRedisPort),
			Username: GetPropStr(PropRedisUsername),
			Password: GetPropStr(PropRedisPassword),
			Db:       GetPropInt(PropRedisDatabas),
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
		return nil, fmt.Errorf("ping redis failed, %w", cmd.Err())
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

func RedisBootstrap(rail Rail) error {
	if _, e := InitRedisFromProp(rail); e != nil {
		return fmt.Errorf("failed to establish connection to Redis, %w", e)
	}
	AddHealthIndicator(HealthIndicator{
		Name: "Redis Component",
		CheckHealth: func(rail Rail) bool {
			cmd := GetRedis().Ping()
			if err := cmd.Err(); err != nil {
				rail.Errorf("Redis ping failed, %v", err)
				return false
			}
			return true
		},
	})
	return nil
}

func RedisBootstrapCondition(rail Rail) (bool, error) {
	return IsRedisEnabled(), nil
}
