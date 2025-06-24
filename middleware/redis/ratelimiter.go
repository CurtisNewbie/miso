package redis

import (
	"time"

	"github.com/go-redis/redis_rate/v7"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	name     string
	limiter  *redis_rate.Limiter
	max      int64
	interval time.Duration
}

func NewRateLimiter(name string, max int64, interval time.Duration) *rateLimiter {
	limiter := redis_rate.NewLimiter(GetRedis())
	limiter.Fallback = rate.NewLimiter(rate.Inf, 0) // permits all if redis is not available
	return &rateLimiter{
		name:     name,
		limiter:  limiter,
		max:      max,
		interval: interval,
	}
}

func (r *rateLimiter) Acquire() bool {
	_, _, ok := r.limiter.Allow(r.name, r.max, r.interval)
	return ok
}
