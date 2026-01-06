package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/strutil"
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
			Address:         miso.GetPropStr(PropRedisAddress),
			Port:            miso.GetPropStr(PropRedisPort),
			Username:        miso.GetPropStr(PropRedisUsername),
			Password:        miso.GetPropStr(PropRedisPassword),
			Db:              miso.GetPropInt(PropRedisDatabase),
			MaxConnPoolSize: miso.GetPropInt(PropRedisMaxPoolSize),
			MinIdleConns:    miso.GetPropInt(PropRedisMinIdleConns),
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

	if p.MaxConnPoolSize == 0 {
		p.MaxConnPoolSize = async.CalcPoolSize(10, 64, -1)
	}

	rail.Infof("Connecting to redis '%v:%v', database: %v, pool_size: %v, min_idle_conns: %v", p.Address, p.Port, p.Db,
		p.MaxConnPoolSize, p.MinIdleConns)

	var rdb *redis.Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", p.Address, p.Port),
		Password:     p.Password,
		DB:           p.Db,
		PoolSize:     p.MaxConnPoolSize,
		MinIdleConns: p.MinIdleConns,
	})

	cmd := rdb.Ping(rail.Context())
	if cmd.Err() != nil {
		return nil, errs.Wrapf(cmd.Err(), "ping redis failed")
	}

	if miso.GetPropBool(PropRedisWithTimingHook) {
		threshold := miso.GetPropDuration(PropRedisSlowLogThreshold)
		rdb.AddHook(timingHook{threshold: threshold})
		rail.Infof("Added TimingHook for Redis Client, threshold: %v", threshold)
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

// Initialize redis client from configuration
//
// If redis client has been initialized, current func call will be ignored.
func InitRedisFromProp(rail miso.Rail) (*redis.Client, error) {
	return module().initFromProp(rail)
}

type RedisConnParam struct {
	Address         string
	Port            string
	Username        string
	Password        string
	Db              int
	MaxConnPoolSize int
	MinIdleConns    int
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
		return errs.Wrapf(e, "failed to establish connection to Redis")
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
	Headers map[string]string `json:"headers"`
	Payload T                 `json:"payload"`
}

type rtopic[T any] struct {
	topic string
}

func (p *rtopic[T]) Subscribe(pool async.AsyncPool, handler func(rail miso.Rail, evt T) error) error {

	pubsub := GetRedis().Subscribe(context.Background(), p.topic)
	miso.AddShutdownHook(func() { pubsub.Close() })

	ch := pubsub.Channel()
	go func() {
		for m := range ch {
			rail := miso.EmptyRail()
			pm, err := json.SParseJsonAs[rtopicMessage[T]](m.Payload)
			if err != nil {
				rail.Errorf("Failed to handle redis channle message, topic: %v, %v", p.topic, err)
				continue
			}
			rail = miso.LoadPropagationKeysFromHeaders(rail, pm.Headers)
			rail.Infof("Receive redis channel message, topic: %v, payload: %#v", p.topic, pm.Payload)

			// redis subscription cannot be blocked for more than 30s, have to handle these asynchronously
			pool.Go(func() {
				if err := handler(rail, pm.Payload); err != nil {
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
	return errs.Wrap(GetRedis().Publish(rail.Context(), p.topic, ms).Err())
}

func NewTopic[T any](topic string) *rtopic[T] {
	return &rtopic[T]{
		topic: topic,
	}
}

type timingHook struct {
	threshold time.Duration
}

func (t timingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (t timingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		defer func() {
			took := time.Since(start)
			if took > t.threshold && !strutil.EqualAnyStr(cmd.Name(), "blpop", "blmpop", "blmove", "brpop", "brpoplpush") {
				miso.NewRail(ctx).Warnf("Slow Redis command, %v, took: %v", cmd.String(), took)
			} else if miso.IsDebugLevel() {
				miso.NewRail(ctx).Debugf("Redis command, %v, took: %v", cmd.String(), took)
			} else if !miso.IsProdMode() && cmd.Name() != "ping" {
				miso.NewRail(ctx).Infof("Redis command, %v, took: %v", cmd.String(), took)
			}
		}()
		return next(ctx, cmd)
	}
}

func (t timingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}
