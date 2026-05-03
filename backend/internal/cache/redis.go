package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type Redis[T any] struct {
	client *redis.Client
	prefix string // for name of the cache, we are using two caches here, price cache and forex cache
	ttl    time.Duration
	sf     singleflight.Group // allows to combine concurrent calls for same key into single execution , example to a new stock - 10 concurrent calls, only one will be processed, rest can access from cache
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

	// If N goroutines all miss price at the same instant, only the first one runs fn, the rest block and receive the same result
	// for example -> if more than one client is asking price for JIo - and its a cache miss, we don't want them all to load and store in cache
	// we will just allow the first to process and the rest to check from cache
	res, err, _ := c.sf.Do(key, func() (any, error) {
		if v, ok, err := c.read(ctx, full); err == nil && ok {
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
	})
	if err != nil {
		return zero, err
	}
	return res.(T), nil
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
		log.Printf("redis %s: bad payload %v", full, jerr)
		return zero, false, nil
	}
	return v, true, nil
}
