package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/redis/go-redis/v9"
)

const (
	Nil = redis.Nil
)

func init() {
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap Redis",
		Bootstrap: redisBootstrap,
		Condition: redisBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

var module = miso.InitAppModuleFunc(newModule)

type redisModule struct {
	mu     *sync.RWMutex
	client *redis.Client
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
	scmd := m.redis().Get(context.Background(), key)
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
			Address:  miso.GetPropStr(PropRedisAddress),
			Port:     miso.GetPropStr(PropRedisPort),
			Username: miso.GetPropStr(PropRedisUsername),
			Password: miso.GetPropStr(PropRedisPassword),
			Db:       miso.GetPropInt(PropRedisDatabase),
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

	cmd := rdb.Ping(rail.Context())
	if cmd.Err() != nil {
		return nil, miso.WrapErrf(cmd.Err(), "ping redis failed")
	}

	rail.Info("Redis connection initialized")
	m.client = rdb
	return rdb, nil
}

func (m *redisModule) addHealthIndicator() {
	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "Redis Component",
		CheckHealth: func(rail miso.Rail) bool {
			cmd := m.redis().Ping(rail.Context())
			if err := cmd.Err(); err != nil {
				rail.Errorf("Redis ping failed, %v", err)
				return false
			}
			return true
		},
	})
}

func newModule() *redisModule {
	return &redisModule{
		mu: &sync.RWMutex{},
	}
}

// Get Redis client
//
// Must call InitRedis(...) method before this method.
func GetRedis() *redis.Client {
	return module().redis()
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

func redisBootstrap(rail miso.Rail) error {
	m := module()
	if _, e := m.initFromProp(rail); e != nil {
		return miso.WrapErrf(e, "failed to establish connection to Redis")
	}
	m.addHealthIndicator()
	redis.SetLogger(redisLogger{})
	return nil
}

func redisBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropRedisEnabled), nil
}

func IsNil(err error) bool {
	return errors.Is(err, Nil)
}

type redisLogger struct {
}

func (r redisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	miso.NewRail(ctx).Infof(format, v...)
}

type rtopicMessage[T any] struct {
	Headers map[string]string
	Payload T
}

type rtopic[T any] struct {
	topic      string
	pubsub     *redis.PubSub
	pubsubOnce *sync.Once
	pool       *util.AsyncPool
	handler    func(rail miso.Rail, t T) error
}

func (p *rtopic[T]) initPubsub() {
	p.pubsubOnce.Do(func() {
		p.pubsub = GetRedis().Subscribe(context.Background(), p.topic)
		miso.AddShutdownHook(func() {
			p.pubsub.Close()
		})
	})
}

func (p *rtopic[T]) Subscribe() error {
	p.initPubsub()
	ch := p.pubsub.Channel()
	go func() {
		for m := range ch {
			rail := miso.EmptyRail()
			pm, err := json.SParseJsonAs[rtopicMessage[T]](m.Payload)
			if err != nil {
				rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
				continue
			}
			rail = miso.LoadPropagationKeysFromHeaders(rail, pm.Headers)
			rail.Debugf("Receive redis channel message, topic: %v", p.topic)

			// redis subscription cannot be blocked for more than 30s, have to handle these asynchronously
			util.SubmitAsync(p.pool, func() (any, error) {
				return nil, p.handler(rail, pm.Payload)
			}).Then(func(a any, err error) {
				if err != nil {
					rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
				}
			})
		}
	}()
	return nil
}

func (p *rtopic[T]) Publish(rail miso.Rail, t T) error {
	m := rtopicMessage[T]{
		Headers: miso.BuildTraceHeadersStr((rail)),
		Payload: t,
	}
	ms, err := json.SWriteJson(m)
	if err != nil {
		return err
	}
	return miso.WrapErr(GetRedis().Publish(rail.Context(), p.topic, ms).Err())
}

func NewTopic[T any](topic string, pool *util.AsyncPool, handler func(rail miso.Rail, evt T) error) *rtopic[T] {
	return &rtopic[T]{
		topic:      topic,
		pubsubOnce: &sync.Once{},
		pool:       pool,
		handler:    handler,
	}
}
