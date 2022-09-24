package util

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bsm/redislock"
	"github.com/curtisnewbie/gocommon/config"
)

type LRunnable func() any

// Check whether the error is 'redislock.ErrNotObtained'
func IsLockNotObtained(err error) bool {
	return err == redislock.ErrNotObtained
}

// Obtain a locker
func ObtainRLocker() *redislock.Client {
	return redislock.New(config.GetRedis())
}

/*
	Lock and run the runnable.
	The maximum time wait for the lock is 10 min.
	May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func LockRun(key string, runnable LRunnable) (any, error) {
	return TimedLockRun(key, 10*time.Minute, runnable)
}

/*
	Lock and run the runnable.
	The ttl is the maximum time wait for the lock.
	May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func TimedLockRun(key string, ttl time.Duration, runnable LRunnable) (any, error) {
	locker := ObtainRLocker()
	lock, err := locker.Obtain(key, ttl, nil)

	if err != nil {
		return nil, err
	}
	log.Infof("Obtained lock for key '%s'", key)

	defer func() {
		lock.Release()
		log.Infof("Released lock for key '%s'", key)
	}()

	return runnable(), nil
}
