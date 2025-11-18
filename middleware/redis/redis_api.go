package redis

import (
	"errors"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/redis/go-redis/v9"
)

func Expire(rail miso.Rail, key string, exp time.Duration) (bool, error) {
	c := GetRedis().Expire(rail.Context(), key, exp)
	v, err := c.Result()
	if err == nil {
		return v, nil
	}
	if !errors.Is(err, redis.Nil) {
		return false, errs.Wrap(err)
	}
	return false, nil
}

func Exists(rail miso.Rail, key string) (bool, error) {
	c := GetRedis().Exists(rail.Context(), key)
	v, err := c.Result()
	if err == nil {
		return v > 0, nil
	}
	if !errors.Is(err, redis.Nil) {
		return false, errs.Wrap(err)
	}
	return false, nil
}

func Get(rail miso.Rail, key string) (string, bool, error) {
	c := GetRedis().Get(rail.Context(), key)
	v, err := c.Result()
	if err == nil {
		return v, true, nil
	}
	if errors.Is(err, redis.Nil) {
		return v, false, nil
	}
	return v, false, errs.Wrap(err)
}

func GetJson[T any](rail miso.Rail, key string) (T, bool, error) {
	var t T
	v, ok, err := Get(rail, key)
	if err != nil || !ok {
		return t, ok, err
	}
	t, err = json.SParseJsonAs[T](v)
	return t, true, err
}

func Set(rail miso.Rail, key string, val any, exp time.Duration) error {
	c := GetRedis().Set(rail.Context(), key, val, exp)
	err := c.Err()
	if err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func SetNX(rail miso.Rail, key string, val any, exp time.Duration) (bool, error) {
	c := GetRedis().SetNX(rail.Context(), key, val, exp)
	v, err := c.Result()
	return v, errs.Wrap(err)
}

func SetJson(rail miso.Rail, key string, val any, exp time.Duration) error {
	s, err := json.SWriteJson(val)
	if err != nil {
		return err
	}
	return Set(rail, key, s, exp)
}

func SetNXJson(rail miso.Rail, key string, val any, exp time.Duration) (bool, error) {
	s, err := json.SWriteJson(val)
	if err != nil {
		return false, err
	}
	return SetNX(rail, key, s, exp)
}

func Scan(rail miso.Rail, pat string, scanLimit int64, f func(key string) error) error {
	cmd := GetRedis().Scan(rail.Context(), 0, pat, scanLimit)
	if cmd.Err() != nil {
		return errs.Wrapf(cmd.Err(), "failed to scan redis with pattern '%v'", pat)
	}

	iter := cmd.Iterator()
	for iter.Next(rail.Context()) {
		if iter.Err() != nil {
			return errs.Wrapf(iter.Err(), "failed to iterate using scan, pattern: '%v'", pat)
		}
		if err := f(iter.Val()); err != nil {
			return err
		}
	}
	return nil
}

func Incr(rail miso.Rail, key string) (after int64, er error) {
	c := GetRedis().Incr(rail.Context(), key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, errs.Wrap(err)
	}
	return v, err
}

func Decr(rail miso.Rail, key string) (after int64, er error) {
	c := GetRedis().Decr(rail.Context(), key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, errs.Wrap(err)
	}
	return v, err
}

func IncrBy(rail miso.Rail, key string, v int64) (after int64, er error) {
	c := GetRedis().IncrBy(rail.Context(), key, v)
	v, err := c.Result()
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return v, err
}

func DecrBy(rail miso.Rail, key string, v int64) (after int64, er error) {
	c := GetRedis().DecrBy(rail.Context(), key, v)
	v, err := c.Result()
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return v, err
}

func LPushJson(rail miso.Rail, key string, v any) error {
	s, err := json.SWriteJson(v)
	if err != nil {
		return err
	}
	return LPush(rail, key, s)
}

func RPushJson(rail miso.Rail, key string, v any) error {
	s, err := json.SWriteJson(v)
	if err != nil {
		return err
	}
	return RPush(rail, key, s)
}

func LPush(rail miso.Rail, key string, v any) error {
	c := GetRedis().LPush(rail.Context(), key, v)
	return errs.Wrap(c.Err())
}

func RPush(rail miso.Rail, key string, v any) error {
	c := GetRedis().RPush(rail.Context(), key, v)
	return errs.Wrap(c.Err())
}

func LPop(rail miso.Rail, key string) (string, bool, error) {
	c := GetRedis().LPop(rail.Context(), key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return v, false, nil
		}
		return v, false, errs.Wrap(err)
	}
	return v, true, nil
}

func RPop(rail miso.Rail, key string) (string, bool, error) {
	c := GetRedis().RPop(rail.Context(), key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return v, false, nil
		}
		return v, false, errs.Wrap(err)
	}
	return v, true, nil
}

func LPopJson[T any](rail miso.Rail, key string) (T, bool, error) {
	var t T
	s, ok, err := LPop(rail, key)
	if err != nil || !ok {
		return t, ok, err
	}
	if err := json.SParseJson(s, &t); err != nil {
		return t, true, err
	}
	return t, true, err
}

func RPopJson[T any](rail miso.Rail, key string) (T, bool, error) {
	var t T
	s, ok, err := RPop(rail, key)
	if err != nil || !ok {
		return t, ok, err
	}
	if err := json.SParseJson(s, &t); err != nil {
		return t, true, err
	}
	return t, true, err
}

func BRPop(rail miso.Rail, timeout time.Duration, key string) ([]string, bool, error) {
	c := GetRedis().BRPop(rail.Context(), timeout, key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return v, false, nil
		}
		return v, false, errs.Wrap(err)
	}
	if len(v) > 0 {
		v = v[1:]
	}
	return v, true, nil

}

func BLPop(rail miso.Rail, timeout time.Duration, key string) ([]string, bool, error) {
	c := GetRedis().BRPop(rail.Context(), timeout, key)
	v, err := c.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return v, false, nil
		}
		return v, false, errs.Wrap(err)
	}
	if len(v) > 0 {
		v = v[1:]
	}
	return v, true, nil
}

func BLPopJson[T any](rail miso.Rail, timeout time.Duration, key string) ([]T, bool, error) {
	s, ok, err := BLPop(rail, timeout, key)
	if err != nil || !ok {
		return nil, ok, err
	}
	v := slutil.MapTo[string, T](s, func(j string) T {
		t, perr := json.SParseJsonAs[T](j)
		if perr != nil {
			err = errs.Wrap(perr)
			rail.Errorf("Failed unmarshal BLPOP value to json, '%v', %v", j, err)
		}
		return t
	})
	return v, true, err
}

func BRPopJson[T any](rail miso.Rail, timeout time.Duration, key string) ([]T, bool, error) {
	s, ok, err := BLPop(rail, timeout, key)
	if err != nil || !ok {
		return nil, ok, err
	}
	v := slutil.MapTo[string, T](s, func(j string) T {
		t, perr := json.SParseJsonAs[T](j)
		if perr != nil {
			err = errs.Wrap(perr)
			rail.Errorf("Failed unmarshal BLPOP value to json, '%v', %v", j, err)
		}
		return t
	})
	return v, true, err
}

func Eval(rail miso.Rail, script string, keys []string, args ...interface{}) (any, error) {
	c := GetRedis().Eval(rail.Context(), script, keys, args...)
	v, err := c.Result()
	return v, errs.Wrap(err)
}
