package common

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	HotReadCacheTTL  = 5 * time.Minute
	QueryCacheTTL    = 30 * time.Second
	negativeCacheTTL = 10 * time.Second
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	DepartmentGeneration(ctx context.Context, departmentID string) (string, error)
	BumpDepartment(ctx context.Context, departmentID string) error
}

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(client *redis.Client, prefix string) (*RedisCache, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment query redis client is required")
	}
	if prefix == "" {
		prefix = "appointment:"
	}
	return &RedisCache{client: client, prefix: prefix + "query:"}, nil
}

func (c *RedisCache) key(value string) string { return c.prefix + value }

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, JitteredTTL(ttl)).Err()
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, c.key(key))
	}
	return c.client.Del(ctx, values...).Err()
}

func (c *RedisCache) DepartmentGeneration(ctx context.Context, departmentID string) (string, error) {
	key := c.key("department:" + departmentID + ":generation")
	value, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		if setErr := c.client.SetNX(ctx, key, "1", 0).Err(); setErr != nil {
			return "", setErr
		}
		value, err = c.client.Get(ctx, key).Result()
	}
	return value, err
}

func (c *RedisCache) BumpDepartment(ctx context.Context, departmentID string) error {
	return c.client.Incr(ctx, c.key("department:"+departmentID+":generation")).Err()
}

func JitteredTTL(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	// 增加 0%～20% 的随机抖动，避免大量缓存同时过期。
	return base + time.Duration(rand.Int64N(int64(base/5+1)))
}

type flightCall struct {
	done  chan struct{}
	value any
	err   error
}

type FlightGroup struct {
	mu    sync.Mutex
	calls map[string]*flightCall
}

func (g *FlightGroup) do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*flightCall)
	}
	if call, exists := g.calls[key]; exists {
		g.mu.Unlock()
		<-call.done
		return call.value, call.err
	}
	call := &flightCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	call.value, call.err = fn()
	close(call.done)
	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()
	return call.value, call.err
}

type cachedValue[T any] struct {
	Found bool `json:"found"`
	Value T    `json:"value"`
}

func LoadCached[T any](
	ctx context.Context,
	cache Cache,
	flights *FlightGroup,
	key string,
	ttl time.Duration,
	loader func() (T, bool, error),
) (T, bool, error) {
	var zero T
	if cache == nil {
		return loader()
	}
	if data, found, err := cache.Get(ctx, key); err == nil && found {
		var entry cachedValue[T]
		if json.Unmarshal(data, &entry) == nil {
			return entry.Value, entry.Found, nil
		}
	}

	value, err := flights.do(key, func() (any, error) {
		if data, found, getErr := cache.Get(ctx, key); getErr == nil && found {
			var entry cachedValue[T]
			if json.Unmarshal(data, &entry) == nil {
				return entry, nil
			}
		}
		loaded, found, loadErr := loader()
		if loadErr != nil {
			return cachedValue[T]{}, loadErr
		}
		entry := cachedValue[T]{Found: found, Value: loaded}
		data, marshalErr := json.Marshal(entry)
		if marshalErr == nil {
			entryTTL := ttl
			if !found {
				entryTTL = negativeCacheTTL
			}
			_ = cache.Set(ctx, key, data, entryTTL)
		}
		return entry, nil
	})
	if err != nil {
		return zero, false, err
	}
	entry := value.(cachedValue[T])
	return entry.Value, entry.Found, nil
}
