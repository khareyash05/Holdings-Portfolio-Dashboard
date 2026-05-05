package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis[T any] struct {
	client *redis.Client
	prefix string // for name of the cache, we are using two caches here, price cache and forex cache
	ttl    time.Duration
}

func NewRedis[T any](client *redis.Client, prefix string, ttl time.Duration) *Redis[T] {
	return &Redis[T]{client: client, prefix: prefix, ttl: ttl}
}

func (c *Redis[T]) key(k string) string {
	return c.prefix + ":" + k
}

func (c *Redis[T]) GetOrLoad(ctx context.Context, key string, loader func() (T, error)) (T, error) {
	var zero T
	full := c.key(key)

	if v, ok, err := c.read(ctx, full); err != nil {
		log.Printf("redis get %s: %v", full, err)
		return loader()
	} else if ok {
		return v, nil
	}

	v, err := loader()
	if err != nil {
		return zero, err
	}
	if buf, jerr := json.Marshal(v); jerr == nil {
		if serr := c.client.Set(ctx, full, buf, c.ttl).Err(); serr != nil {
			log.Printf("redis set %s: %v ", full, serr)
		}
	}
	return v, nil
}

func (c *Redis[T]) Get(ctx context.Context, key string) (T, bool, error) {
	return c.read(ctx, c.key(key))
}

func (c *Redis[T]) Set(ctx context.Context, key string, v T) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), buf, c.ttl).Err()
}

func (c *Redis[T]) read(ctx context.Context, full string) (T, bool, error) {
	var zero T
	raw, err := c.client.Get(ctx, full).Bytes()
	if errors.Is(err, redis.Nil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var v T
	if jerr := json.Unmarshal(raw, &v); jerr != nil {
		return zero, false, nil
	}
	return v, true, nil
}
