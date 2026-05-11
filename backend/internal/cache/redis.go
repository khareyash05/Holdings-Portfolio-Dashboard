package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/clients"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
)

type Redis[T any] struct {
	client  *redis.Client
	prefix  string // for name of the cache, we are using two caches here, price cache and forex cache
	ttl     time.Duration
	breaker *gobreaker.CircuitBreaker[[]byte] // added a circuit breaker for redis so that our service don't call dead cache(in this case all the data will come via db)
}

func NewRedis[T any](client *redis.Client, prefix string, ttl time.Duration) *Redis[T] {
	return &Redis[T]{client: client, prefix: prefix, ttl: ttl, breaker: clients.NewBreaker[[]byte]("redis:" + prefix)}
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
		if _, serr := c.breaker.Execute(func() ([]byte, error) {
			return nil, c.client.Set(ctx, full, buf, c.ttl).Err()
		}); serr != nil {
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
	_, err = c.breaker.Execute(func() ([]byte, error) {
		return nil, c.client.Set(ctx, c.key(key), buf, c.ttl).Err()
	})
	return err
}

func (c *Redis[T]) read(ctx context.Context, full string) (T, bool, error) {
	var zero T
	raw, err := c.breaker.Execute(func() ([]byte, error) {
		b, gerr := c.client.Get(ctx, full).Bytes()
		if errors.Is(gerr, redis.Nil) {
			return nil, nil
		}
		return b, gerr
	})
	if err != nil {
		return zero, false, err
	}
	if raw == nil {
		return zero, false, nil
	}
	var v T
	if jerr := json.Unmarshal(raw, &v); jerr != nil {
		return zero, false, nil
	}
	return v, true, nil
}
