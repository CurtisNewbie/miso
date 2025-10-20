package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/errs"
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
func RLockRun[T any](rail miso.Rail, key string, runnable LRunnable[T], op ...rLockOption) (T, error) {
	var t T

	lock := NewRLock(rail, key)
	if err := lock.Lock(op...); err != nil {
		return t, errs.Wrapf(err, "failed to obtain lock, key: %v", key)
	}
	defer lock.Unlock()

	return runnable()
}

/*
Lock and run the runnable using Redis

The maximum time wait for the lock is 1s, retry every 5ms.

May return 'redislock.ErrNotObtained' when it fails to obtain the lock.
*/
func RLockExec(ec miso.Rail, key string, runnable Runnable, op ...rLockOption) error {
	_, e := RLockRun(ec, key, func() (any, error) {
		return nil, runnable()
	}, op...)
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
		r.backoffSteps = int(int64(config.BackoffDuration) / int64(r.backoffWindow))
	} else if config.BackoffDuration > 0 {
		r.backoffSteps = 1
		r.backoffWindow = config.BackoffDuration
	}
	return &r
}

// Try Lock.
//
// Use [WithBackoff] to modify default configuration.
func (r *RLock) TryLock(op ...rLockOption) (locked bool, err error) {
	err = r.Lock(op...)
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type rLockOption func(r *RLock)

func WithBackoff(backoff time.Duration) rLockOption {
	return func(r *RLock) {

		if backoff > r.backoffWindow {
			// this is merely an approximate
			r.backoffSteps = int(int64(backoff) / int64(r.backoffWindow))
		} else if backoff > 0 {
			r.backoffSteps = 1
			r.backoffWindow = backoff
		} else {
			r.backoffSteps = defaultBackoffSteps // default backoff steps
		}

		r.rail.Tracef("Updated RLock for key: '%v' to using backoff: %v", r.key, backoff)
	}
}

// Acquire lock.
//
// Use [WithBackoff] to modify default configuration.
func (r *RLock) Lock(op ...rLockOption) error {
	for _, fn := range op {
		fn(r)
	}

	if miso.IsTraceLevel() {
		r.rail.Tracef("RLock locking for key: '%v', with backoffWindow: %v, backoffSteps: %v (around: %v)", r.key, r.backoffWindow, r.backoffSteps,
			time.Duration(r.backoffSteps*int(r.backoffWindow)))
	}

	rlocker := ObtainRLocker()
	lock, err := rlocker.Obtain(context.Background(), r.key, lockLeaseTime, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(r.backoffWindow), r.backoffSteps),
	})
	if err != nil {
		return errs.Wrapf(err, "failed to obtain lock, key: %v", r.key)
	}
	lockStart := time.Now()
	r.lock = lock
	r.rail.Debugf("Obtained lock for key '%s'", r.key)

	refreshCtx, cancel := context.WithCancel(context.Background())
	r.cancelRefresher = cancel

	go func(ctx context.Context) {
		ticker := time.NewTicker(lockRefreshTime)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := lock.Refresh(context.Background(), lockLeaseTime, nil); err != nil {
					if errors.Is(err, redislock.ErrNotObtained) {
						return
					}
					r.rail.Warnf("Failed to refresh RLock for '%v', %v", r.key, err)
				} else {
					r.rail.Infof("Refreshed rlock for '%v', held_lock_for: %v", r.key, time.Since(lockStart))
				}
			case <-ctx.Done():
				r.rail.Debugf("RLock Refresher cancelled for '%v'", r.key)
				return
			}
		}
	}(refreshCtx)

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
		err := r.lock.Release(context.Background())
		if err != nil {
			if errors.Is(err, redislock.ErrLockNotHeld) {
				return nil
			}
			r.rail.Errorf("Failed to release lock for key '%s', err: %v", r.key, err)
			return err
		} else {
			r.rail.Debugf("Released lock for key '%s'", r.key)
		}
		r.lock = nil
	}

	return nil
}
