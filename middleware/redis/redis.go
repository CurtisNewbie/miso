package redis

import (
	"errors"
	"fmt"
	"sync"

	"github.com/curtisnewbie/miso/miso"
	"github.com/go-redis/redis"
)

const (
	Nil = redis.Nil
)

func init() {
	miso.SetDefProp(PropRedisEnabled, false)
	miso.SetDefProp(PropRedisAddress, "localhost")
	miso.SetDefProp(PropRedisPort, 6379)
	miso.SetDefProp(PropRedisUsername, "")
	miso.SetDefProp(PropRedisPassword, "")
	miso.SetDefProp(PropRedisDatabase, 0)

	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap Redis",
		Bootstrap: redisBootstrap,
		Condition: redisBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

var appModule, module = miso.InitAppModuleFunc(func(app *miso.MisoApp) *redisModule {
	return newModule(app)
})

type redisModule struct {
	mu     *sync.RWMutex
	client *redis.Client
	app    *miso.MisoApp
	conf   *miso.AppConfig
}

func (m *redisModule) redis() *redis.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil {
		panic("Redis Connection hasn't been initialized yet")
	}
	return m.client
}

func (m *redisModule) getStr(key string) (string, error) {
	scmd := m.redis().Get(key)
	e := scmd.Err()
	if e != nil {
		if errors.Is(e, redis.Nil) {
			return "", nil
		}
		return "", e
	}
	return scmd.Val(), nil
}

func (m *redisModule) initFromProp(rail miso.Rail) (*redis.Client, error) {
	return m.init(
		rail,
		RedisConnParam{
			Address:  m.conf.GetPropStr(PropRedisAddress),
			Port:     m.conf.GetPropStr(PropRedisPort),
			Username: m.conf.GetPropStr(PropRedisUsername),
			Password: m.conf.GetPropStr(PropRedisPassword),
			Db:       m.conf.GetPropInt(PropRedisDatabase),
		})
}

func (m *redisModule) init(rail miso.Rail, p RedisConnParam) (*redis.Client, error) {
	m.mu.RLock()
	if m.client != nil {
		m.mu.RUnlock()
		return m.client, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return m.client, nil
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
	m.client = rdb
	return rdb, nil
}

func (m *redisModule) addHealthIndicator() {
	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "Redis Component",
		CheckHealth: func(rail miso.Rail) bool {
			cmd := m.redis().Ping()
			if err := cmd.Err(); err != nil {
				rail.Errorf("Redis ping failed, %v", err)
				return false
			}
			return true
		},
	})
}

func newModule(app *miso.MisoApp) *redisModule {
	return &redisModule{
		mu:   &sync.RWMutex{},
		conf: app.Config(),
		app:  app,
	}
}

// Get Redis client
//
// Must call InitRedis(...) method before this method.
func GetRedis() *redis.Client {
	return module().redis()
}

func AppGetRedis(app *miso.MisoApp) *redis.Client {
	return appModule(app).redis()
}

// Get String
func GetStr(key string) (string, error) {
	return module().getStr(key)
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
func InitRedisFromProp(rail miso.Rail) (*redis.Client, error) {
	return module().initFromProp(rail)
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
func InitRedis(rail miso.Rail, p RedisConnParam) (*redis.Client, error) {
	return module().init(rail, p)
}

func redisBootstrap(app *miso.MisoApp, rail miso.Rail) error {
	m := appModule(app)
	if _, e := m.initFromProp(rail); e != nil {
		return fmt.Errorf("failed to establish connection to Redis, %w", e)
	}
	m.addHealthIndicator()
	return nil
}

func redisBootstrapCondition(app *miso.MisoApp, rail miso.Rail) (bool, error) {
	return app.Config().GetPropBool(PropRedisEnabled), nil
}

func IsNil(err error) bool {
	return errors.Is(err, Nil)
}
