package redis

import (
	"time"

	"github.com/curtisnewbie/gocommon/common"

	"github.com/bsm/redislock"
)

type Runnable func() error
type LRunnable func() (any, error)

var (
	lock_ttl_min int = 10
)

// Check whether the error is 'redislock.ErrNotObtained'
func IsRLockNotObtainedErr(err error) bool {
	return err == redislock.ErrNotObtained
}

// Obtain a locker
func ObtainRLocker() *redislock.Client {
	return redislock.New(GetRedis())
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1 min.
May return 'redislock:.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockRun(ec common.ExecContext, key string, runnable LRunnable) (any, error) {
	return TimedRLockRun(ec, key, 1*time.Minute, runnable)
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1 min.
May return 'redislock:.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockExec(ec common.ExecContext, key string, runnable Runnable) error {
	_, e := TimedRLockRun(ec, key, 1*time.Minute, func() (any, error) {
		return nil, runnable()
	})
	return e
}

/*
Lock and run the runnable using Redis

The ttl is the maximum time wait for the lock.
May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func TimedRLockRun(ec common.ExecContext, key string, ttl time.Duration, runnable LRunnable) (any, error) {
	locker := ObtainRLocker()
	lock, err := locker.Obtain(key, time.Duration(lock_ttl_min)*time.Minute, &redislock.Options{
		RetryStrategy: redislock.LinearBackoff(ttl),
	})

	if err != nil {
		return nil, err
	}

	ec.Log.Debugf("Obtained lock for key '%s'", key)

	defer func() {
		re := lock.Release()

		if re != nil {
			ec.Log.Errorf("Failed to release lock for key '%s', err: %v", key, re)
		} else {
			ec.Log.Debugf("Released lock for key '%s'", key)
		}
	}()

	return runnable()
}
