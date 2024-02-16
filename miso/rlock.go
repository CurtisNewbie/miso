package miso

import (
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"
)

type Runnable func() error
type LRunnable[T any] func() (T, error)

const (
	defaultBackoffSteps = 1000 // default backoff steps, 1000 (5ms) = 5s
)

var (
	lockLeaseTime   = time.Duration(30_000) * time.Millisecond
	lockRefreshTime = time.Duration(10_000) * time.Millisecond
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

The maximum time wait for the lock is 1s, retry every 5ms.

May return 'redislock:ErrNotObtained' when it fails to obtain the lock.
*/
func RLockRun[T any](rail Rail, key string, runnable LRunnable[T]) (T, error) {
	var t T

	lock := NewRLock(rail, key)
	if err := lock.Lock(); err != nil {
		return t, fmt.Errorf("failed to obtain lock, key: %v, %w", key, err)
	}
	defer lock.Unlock()

	return runnable()
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1s, retry every 5ms.

May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockExec(ec Rail, key string, runnable Runnable) error {
	_, e := RLockRun(ec, key, func() (any, error) {
		return nil, runnable()
	})
	return e
}

// RLock
type RLock struct {
	rail            Rail
	key             string
	cancelRefresher func()
	lock            *redislock.Lock
	backoffWindow   time.Duration
	backoffSteps    int
}

// RLock Configuration.
type RLockConfig struct {
	// Backoff duration.
	//
	// This is not an exact configuration. Linear back off strategy is used with a window size of 5ms, which means RLock will attempt to acquire the lock every 5ms.
	//
	// The number of times we may attempt to acquire the lock is called steps, which is by default 200 (that's 1s = 200 * 5ms).
	// When BackoffDuration is provided, this duration is divided by 5ms to convert to steps and then used by RLock.
	BackoffDuration time.Duration
}

// Create new RLock with default backoff configuration (5ms backoff window, 1000 attempts, i.e., retry for 5s).
func NewRLock(rail Rail, key string) *RLock {
	return NewCustomRLock(rail, key, RLockConfig{})
}

// Create new RLock with default backoff configuration (5ms backoff window, 1000 attempts, i.e., retry for 5s).
func NewRLockf(rail Rail, keyPattern string, args ...any) *RLock {
	return NewCustomRLock(rail, fmt.Sprintf(keyPattern, args...), RLockConfig{})
}

// Create customized RLock.
func NewCustomRLock(rail Rail, key string, config RLockConfig) *RLock {
	r := RLock{
		rail:            rail,
		key:             key,
		cancelRefresher: nil,
		lock:            nil,
		backoffWindow:   5 * time.Millisecond,
		backoffSteps:    defaultBackoffSteps,
	}

	if config.BackoffDuration > r.backoffWindow {
		// this is merely an approximate
		nsteps := int64(config.BackoffDuration) / int64(r.backoffWindow)
		r.backoffSteps = int(nsteps)
		rail.Tracef("Update backoff steps to %v", nsteps)
	}

	rail.Tracef("Created RLock for key: '%v', with backoffWindow: %v, backoffSteps: %v", r.key, r.backoffWindow, r.backoffSteps)
	return &r
}

// Acquire lock.
func (r *RLock) Lock() error {
	rlocker := ObtainRLocker()
	lock, err := rlocker.Obtain(r.key, lockLeaseTime, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(r.backoffWindow), r.backoffSteps),
	})
	if err != nil {
		return fmt.Errorf("failed to obtain lock, key: %v, %w", r.key, err)
	}
	r.lock = lock
	r.rail.Tracef("Obtained lock for key '%s'", r.key)

	cancelChan := make(chan struct{}, 1)
	r.cancelRefresher = func() {
		cancelChan <- struct{}{}
	}

	go func(rail Rail) {
		ticker := time.NewTicker(lockRefreshTime)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := lock.Refresh(lockLeaseTime, nil); err != nil {
					rail.Warnf("Failed to refresh RLock for '%v', %v", r.key, err)
					if errors.Is(err, redislock.ErrNotObtained) {
						return
					}
				} else {
					rail.Infof("Refreshed rlock for '%v'", r.key)
				}
			case <-cancelChan:
				rail.Tracef("RLock Refresher cancelled for '%v'", r.key)
				return
			}
		}
	}(r.rail.NextSpan())

	return nil
}

// Attempt to Unlock.
//
// If the lock is not obtained, method call will be ignored.
func (r *RLock) Unlock() error {
	if r.cancelRefresher != nil {
		r.cancelRefresher()
	}

	if r.lock != nil {
		err := r.lock.Release()
		if err != nil {
			r.rail.Errorf("Failed to release lock for key '%s', err: %v", r.key, err)
			return err
		} else {
			r.rail.Tracef("Released lock for key '%s'", r.key)
		}
		r.lock = nil
	}

	return nil
}
