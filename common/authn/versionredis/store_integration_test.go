package versionredis

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func TestStoreAdvanceAuthorizationVersionIntegration(t *testing.T) {
	addr := os.Getenv("AUTHORIZATION_VERSION_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("AUTHORIZATION_VERSION_REDIS_TEST_ADDR is not configured")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr, Password: os.Getenv("AUTHORIZATION_VERSION_REDIS_TEST_PASSWORD"),
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}

	prefix := "identity:test:authorization-version:" + uuid.NewString() + ":"
	store, err := NewStore(client, prefix)
	if err != nil {
		t.Fatal(err)
	}
	key := prefix + "account-1"
	t.Cleanup(func() { _ = client.Del(ctx, key).Err() })

	tests := []struct {
		name        string
		version     int64
		wantUpdated bool
	}{
		{name: "create", version: 5, wantUpdated: true},
		{name: "duplicate", version: 5, wantUpdated: false},
		{name: "stale", version: 4, wantUpdated: false},
		{name: "newer", version: 6, wantUpdated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := store.AdvanceAuthorizationVersion(ctx, "account-1", tt.version)
			if err != nil {
				t.Fatal(err)
			}
			if updated != tt.wantUpdated {
				t.Fatalf("unexpected update result: got %t want %t", updated, tt.wantUpdated)
			}
		})
	}

	version, err := store.CurrentAuthorizationVersion(ctx, "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if version != 6 {
		t.Fatalf("unexpected final authorization version: got %d want 6", version)
	}
}
