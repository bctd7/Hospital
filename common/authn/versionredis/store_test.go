package versionredis

import (
	"context"
	"strconv"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func TestStoreReadsAndWritesStableAccountKey(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewStore(client, "test:authorization:")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetAuthorizationVersion(context.Background(), "account-1", 7); err != nil {
		t.Fatal(err)
	}
	version, err := store.CurrentAuthorizationVersion(context.Background(), "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if version != 7 || client.lastKey != "test:authorization:account-1" {
		t.Fatalf("unexpected version projection: version=%d key=%q", version, client.lastKey)
	}
}

func TestStoreUsesDefaultPrefix(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewStore(client, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetAuthorizationVersion(context.Background(), "account-1", 1); err != nil {
		t.Fatal(err)
	}
	if client.lastKey != DefaultPrefix+"account-1" {
		t.Fatalf("unexpected key: %q", client.lastKey)
	}
}

type fakeRedisClient struct {
	values  map[string]int64
	lastKey string
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{values: make(map[string]int64)}
}

func (c *fakeRedisClient) Get(_ context.Context, key string) *redis.StringCmd {
	c.lastKey = key
	value, exists := c.values[key]
	if !exists {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(stringValue(value), nil)
}

func (c *fakeRedisClient) Set(_ context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	c.lastKey = key
	version, ok := value.(int64)
	if !ok {
		return redis.NewStatusResult("", nil)
	}
	c.values[key] = version
	return redis.NewStatusResult("OK", nil)
}

func stringValue(value int64) string {
	return strconv.FormatInt(value, 10)
}
