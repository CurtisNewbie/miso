package redis

import (
	"errors"
	"time"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/redis/go-redis/v9"
)

func Expire(rail miso.Rail, key string, exp time.Duration) (bool, error) {
	c := GetRedis().Expire(rail.Context(), key, exp)
	v, err := c.Result()
	if err == nil {
		return v, nil
	}
	if !errors.Is(err, redis.Nil) {
		return false, errs.WrapErr(err)
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
		return false, errs.WrapErr(err)
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
	return v, false, errs.WrapErr(err)
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
		return err
	}
	return nil
}

func SetNX(rail miso.Rail, key string, val any, exp time.Duration) (bool, error) {
	c := GetRedis().SetNX(rail.Context(), key, val, exp)
	return c.Result()
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
		return errs.WrapErrf(cmd.Err(), "failed to scan redis with pattern '%v'", pat)
	}

	iter := cmd.Iterator()
	for iter.Next(rail.Context()) {
		if iter.Err() != nil {
			return errs.WrapErrf(iter.Err(), "failed to iterate using scan, pattern: '%v'", pat)
		}
		if err := f(iter.Val()); err != nil {
			return err
		}
	}
	return nil
}
