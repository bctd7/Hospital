package redisstore

import (
	"context"
	"errors"
	"strconv"
	"testing"

	redis "github.com/redis/go-redis/v9"
)

func TestStoreReadsAndWritesStableAccountKey(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewStore(client, "test:authorization:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceAuthorizationVersion(context.Background(), "account-1", 7); err != nil {
		t.Fatal(err)
	}
	version, err := store.CurrentAuthorizationVersion(context.Background(), "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if version != 7 || client.lastKey != "test:authorization:account-1" {
		t.Fatalf("unexpected stored version: version=%d key=%q", version, client.lastKey)
	}
}

func TestStoreUsesDefaultPrefix(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewStore(client, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceAuthorizationVersion(context.Background(), "account-1", 1); err != nil {
		t.Fatal(err)
	}
	if client.lastKey != DefaultPrefix+"account-1" {
		t.Fatalf("unexpected key: %q", client.lastKey)
	}
}

func TestStoreAdvancesAuthorizationVersionMonotonically(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewStore(client, "test:authorization:")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.AdvanceAuthorizationVersion(context.Background(), "account-1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected missing authorization version to be created")
	}

	updated, err = store.AdvanceAuthorizationVersion(context.Background(), "account-1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("expected duplicate authorization version to be ignored")
	}

	updated, err = store.AdvanceAuthorizationVersion(context.Background(), "account-1", 4)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("expected stale authorization version to be ignored")
	}

	updated, err = store.AdvanceAuthorizationVersion(context.Background(), "account-1", 6)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected newer authorization version to be stored")
	}

	version, err := store.CurrentAuthorizationVersion(context.Background(), "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if version != 6 {
		t.Fatalf("unexpected final authorization version: got %d want 6", version)
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

func (c *fakeRedisClient) Eval(
	_ context.Context,
	_ string,
	keys []string,
	args ...any,
) *redis.Cmd {
	if len(keys) != 1 {
		return redis.NewCmdResult(nil, errors.New("expected exactly one redis key"))
	}
	if len(args) != 1 {
		return redis.NewCmdResult(nil, errors.New("expected exactly one script argument"))
	}

	version, ok := args[0].(int64)
	if !ok {
		return redis.NewCmdResult(nil, errors.New("expected int64 authorization version"))
	}

	key := keys[0]
	c.lastKey = key
	current, exists := c.values[key]
	if exists && current >= version {
		return redis.NewCmdResult(int64(0), nil)
	}

	c.values[key] = version
	return redis.NewCmdResult(int64(1), nil)
}

func stringValue(value int64) string {
	return strconv.FormatInt(value, 10)
}
