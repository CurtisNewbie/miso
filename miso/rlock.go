package miso

import (
	"sync/atomic"
	"time"

	"github.com/bsm/redislock"
)

type Runnable func() error
type LRunnable[T any] func() (T, error)

const (
	defaultBackoffSteps = 200 // default backoff steps, 200 (5ms) = 1s
)

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

The maximum time wait for the lock is 1s, retry every 5ms.

May return 'redislock:ErrNotObtained' when it fails to obtain the lock.
*/
func RLockRun[T any](rail Rail, key string, runnable LRunnable[T]) (T, error) {
	var t T

	lock := NewRLock(rail, key)
	if err := lock.Lock(); err != nil {
		return t, TraceErrf(err, "failed to obtain lock, key: %v", key)
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
	_isLocked       int32
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

// Create new RLock with default backoff configuration (5ms backoff window, 200 attempts, i.e., retry for 1s).
func NewRLock(rail Rail, key string) RLock {
	return NewCustomRLock(rail, key, RLockConfig{})
}

// Create customized RLock.
func NewCustomRLock(rail Rail, key string, config RLockConfig) RLock {
	r := RLock{
		rail:            rail,
		key:             key,
		_isLocked:       0,
		cancelRefresher: nil,
		lock:            nil,
		backoffWindow:   5 * time.Millisecond,
		backoffSteps:    defaultBackoffSteps,
	}

	if config.BackoffDuration > r.backoffWindow {
		// this is merely an approximate
		nsteps := int64(config.BackoffDuration) / int64(r.backoffWindow)
		r.backoffSteps = int(nsteps)
		rail.Debugf("Update backoff steps to %v", nsteps)
	}

	rail.Debugf("Created RLock for key: '%v', with backoffWindow: %v, backoffSteps: %v", r.key, r.backoffWindow, r.backoffSteps)
	return r
}

func (r *RLock) isLocked() bool {
	if r.lock == nil {
		return false
	}
	return atomic.LoadInt32(&r._isLocked) == 1
}

func (r *RLock) setLocked() {
	atomic.StoreInt32(&r._isLocked, 1)
}

func (r *RLock) setUnlocked() {
	atomic.StoreInt32(&r._isLocked, 0)
}

// Acquire lock.
func (r *RLock) Lock() error {
	rlocker := ObtainRLocker()
	lock, err := rlocker.Obtain(r.key, lock_lease_time, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(r.backoffWindow), r.backoffSteps),
	})

	if err != nil {
		return TraceErrf(err, "failed to obtain lock, key: %v", r.key)
	}
	r.rail.Debugf("Obtained lock for key '%s'", r.key)

	r.setLocked()
	r.lock = lock

	subrail, cancel := r.rail.WithCancel()
	r.cancelRefresher = cancel

	go func(subrail Rail) {
		ticker := time.NewTicker(refresh_time)
		for {
			select {
			case <-ticker.C:
				if err := lock.Refresh(lock_lease_time, nil); err != nil {
					subrail.Warnf("Failed to refresh RLock for '%v'", r.key)
				} else {
					subrail.Debugf("Refreshed rlock for '%v'", r.key)
				}
			case <-subrail.Ctx.Done():
				subrail.Debugf("RLock Refresher cancelled for '%v'", r.key)
				return
			}
		}
	}(subrail)

	return nil
}

// Attempt to Unlock.
//
// If the lock is not obtained, method call will be ignored.
func (r *RLock) Unlock() error {
	if !r.isLocked() {
		return nil
	}

	if r.cancelRefresher != nil {
		r.cancelRefresher()
	}

	if r.lock != nil {
		err := r.lock.Release()
		if err != nil {
			r.rail.Errorf("Failed to release lock for key '%s', err: %v", r.key, err)
			return err
		} else {
			r.rail.Debugf("Released lock for key '%s'", r.key)
		}
	}
	r.setUnlocked()
	r.lock = nil
	return nil
}
