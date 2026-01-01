package svc

import (
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/middleware/redis"
)

type redisSvcLock struct {
	r *redis.RLock
}

func (r *redisSvcLock) Lock(lc LockContext) error {
	r.r = redis.NewRLockf(lc.Rail, "miso:svc:%v", lc.App)
	lc.Rail.Info("Obtaining svc lock for migration")
	ok, err := r.r.TryLock(redis.WithBackoff(time.Second * 30))
	if err != nil {
		return err
	}
	if !ok {
		return errs.NewErrf("Failed to obtain svc lock for schema migration, other node undertaking migration, cannot bootstrap yet")
	}
	return nil
}

func (r *redisSvcLock) Unlock() {
	r.r.Unlock()
}

// Use Redis as Locker for schema migration.
func WithRedisLocker() func(*schemaMigrateOptions) {
	return WithLocker(&redisSvcLock{})
}
