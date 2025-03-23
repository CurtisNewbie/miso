package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/curtisnewbie/miso/miso"
)

type Runnable func() error
type LRunnable[T any] func() (T, error)

const (
	defaultBackoffSteps = 6_000 // default backoff steps, 6_000 (5ms) = 30s
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
func RLockRun[T any](rail miso.Rail, key string, runnable LRunnable[T]) (T, error) {
	var t T

	lock := NewRLock(rail, key)
	if err := lock.Lock(); err != nil {
		return t, miso.WrapErrf(err, "failed to obtain lock, key: %v", key)
	}
	defer lock.Unlock()

	return runnable()
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1s, retry every 5ms.

May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockExec(ec miso.Rail, key string, runnable Runnable) error {
	_, e := RLockRun(ec, key, func() (any, error) {
		return nil, runnable()
	})
	return e
}

// RLock
type RLock struct {
	rail            miso.Rail
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
	// The number of times we may attempt to acquire the lock is called steps, which is by default 600 (that's 30s = 600 * 5ms).
	// When BackoffDuration is provided, this duration is divided by 5ms to convert to steps and then used by RLock.
	BackoffDuration time.Duration
}

// Create new RLock with default backoff configuration (5ms backoff window, 6000 attempts, i.e., retry for 30s).
func NewRLock(rail miso.Rail, key string) *RLock {
	return NewCustomRLock(rail, key, RLockConfig{})
}

// Create new RLock with default backoff configuration (5ms backoff window, 6000 attempts, i.e., retry for 30s).
func NewRLockf(rail miso.Rail, keyPattern string, args ...any) *RLock {
	return NewCustomRLock(rail, fmt.Sprintf(keyPattern, args...), RLockConfig{})
}

// Create customized RLock.
func NewCustomRLock(rail miso.Rail, key string, config RLockConfig) *RLock {
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

func (r *RLock) TryLock() (locked bool, err error) {
	err = r.Lock()
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Acquire lock.
func (r *RLock) Lock() error {
	rlocker := ObtainRLocker()
	lock, err := rlocker.Obtain(r.key, lockLeaseTime, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(r.backoffWindow), r.backoffSteps),
	})
	if err != nil {
		return miso.WrapErrf(err, "failed to obtain lock, key: %v", r.key)
	}
	r.lock = lock
	r.rail.Debugf("Obtained lock for key '%s'", r.key)

	srcSpan := r.rail.SpanId()
	refreshCtx, cancel := context.WithCancel(context.Background())
	r.cancelRefresher = cancel

	go func() {
		rail := r.rail.NextSpan()
		ticker := time.NewTicker(lockRefreshTime)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := lock.Refresh(lockLeaseTime, nil); err != nil {
					if errors.Is(err, redislock.ErrNotObtained) {
						return
					}
					rail.Warnf("Failed to refresh RLock for '%v', %v", r.key, err)
				} else {
					rail.Infof("Refreshed rlock for '%v', source span_id: %v", r.key, srcSpan)
				}
			case <-refreshCtx.Done():
				rail.Debugf("RLock Refresher cancelled for '%v'", r.key)
				return
			}
		}
	}()

	return nil
}

// Attempt to Unlock.
//
// If the lock is not obtained, method call will be ignored.
func (r *RLock) Unlock() error {
	if r.lock != nil {
		err := r.lock.Release()
		if err != nil {
			r.rail.Errorf("Failed to release lock for key '%s', err: %v", r.key, err)
			return err
		} else {
			r.rail.Debugf("Released lock for key '%s'", r.key)
		}
		r.lock = nil

		if r.cancelRefresher != nil {
			r.cancelRefresher()
		}
	}
	return nil
}
