package redis

import (
	"context"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/go-redis/redis_rate/v10"
)

type rateLimiter struct {
	name    string
	limiter *redis_rate.Limiter
	max     int
	period  time.Duration
}

func NewRateLimiter(name string, max int, period time.Duration) *rateLimiter {
	limiter := redis_rate.NewLimiter(GetRedis())
	return &rateLimiter{
		name:    name,
		limiter: limiter,
		max:     max,
		period:  period,
	}
}

func (r *rateLimiter) Acquire() (bool, error) {
	res, err := r.limiter.Allow(context.Background(), r.name, redis_rate.Limit{
		Rate:   r.max,
		Burst:  r.max,
		Period: r.period,
	})
	if err != nil {
		return false, miso.WrapErr(err)
	}
	return res.Allowed > 0, nil
}
