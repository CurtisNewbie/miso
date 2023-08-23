package redis

import (
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/core"

	"github.com/bsm/redislock"
)

type Runnable func() error
type LRunnable[T any] func() (T, error)

var (
	lock_lease_time = time.Duration(30_000) * time.Millisecond
	refresh_time    = time.Duration(10_000) * time.Millisecond
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

The maximum time wait for the lock is 1s, retry every 10ms.

May return 'redislock:.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockRun[T any](ec core.Rail, key string, runnable LRunnable[T]) (T, error) {
	var t T
	locker := ObtainRLocker()
	lock, err := locker.Obtain(key, lock_lease_time, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(10*time.Millisecond), 100),
	})

	if err != nil {
		return t, core.TraceErrf(err, "failed to obtain lock, key: %v", key)
	}
	ec.Debugf("Obtained lock for key '%s'", key)

	var isReleased int32 = 0 // 0-locked, 1-released

	// watchdog for the lock
	go func() {
		isReleased := func() bool { return atomic.LoadInt32(&isReleased) == 1 }
		ticker := time.NewTicker(refresh_time)
		for range ticker.C {
			if isReleased() {
				ticker.Stop()
				return
			}

			if err := lock.Refresh(lock_lease_time, nil); err != nil {
				ec.Warnf("Failed to refresh rlock for '%v'", key)
			}
			ec.Debugf("Refreshed rlock for '%v'", key)
		}
	}()

	defer func() {
		atomic.StoreInt32(&isReleased, 1)
		re := lock.Release()

		if re != nil {
			ec.Errorf("Failed to release lock for key '%s', err: %v", key, re)
		} else {
			ec.Debugf("Released lock for key '%s'", key)
		}
	}()

	return runnable()
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1s, retry every 10ms.

May return 'redislock:.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockExec(ec core.Rail, key string, runnable Runnable) error {
	_, e := RLockRun(ec, key, func() (any, error) {
		return nil, runnable()
	})
	return e
}
