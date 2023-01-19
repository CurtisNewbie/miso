package redis

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bsm/redislock"
)

type LRunnable func() (any, error)

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
func RLockRun(key string, runnable LRunnable) (any, error) {
	return TimedRLockRun(key, 1*time.Minute, runnable)
}

/*
	Lock and run the runnable using Redis

	The ttl is the maximum time wait for the lock.
	May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func TimedRLockRun(key string, ttl time.Duration, runnable LRunnable) (any, error) {
	locker := ObtainRLocker()
	lock, err := locker.Obtain(key, ttl, nil)

	if err != nil {
		return nil, err
	}
	log.Infof("Obtained lock for key '%s'", key)

	defer func() {
		re := lock.Release()
		log.Infof("Released lock for key '%s', err: %v", key, re)
	}()

	return runnable()
}
