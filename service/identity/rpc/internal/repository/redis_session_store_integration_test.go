package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"

	"hospital/service/identity/rpc/internal/session"
)

func TestRedisSessionStoreLifecycle(t *testing.T) {
	addr := os.Getenv("IDENTITY_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("IDENTITY_REDIS_TEST_ADDR is not configured")
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr, Password: os.Getenv("IDENTITY_REDIS_TEST_PASSWORD"),
	})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	prefix := "identity:test:refresh:" + uuid.NewString() + ":"
	store, err := NewRedisSessionStore(client, prefix)
	if err != nil {
		t.Fatal(err)
	}

	value := session.Session{
		ID: uuid.NewString(), FamilyID: uuid.NewString(), AccountID: uuid.NewString(),
		TokenHash: "old-hash", AuthorizationVersion: 1, ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := store.Create(ctx, value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Del(ctx, prefix+value.ID).Err() })
	loaded, err := store.Get(ctx, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccountID != value.AccountID || loaded.TokenHash != value.TokenHash {
		t.Fatalf("unexpected stored session: %#v", loaded)
	}

	if err := store.Rotate(ctx, value.ID, "old-hash", "new-hash", 2); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Get(ctx, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TokenHash != "new-hash" || loaded.AuthorizationVersion != 2 {
		t.Fatalf("refresh session was not rotated: %#v", loaded)
	}

	if err := store.Rotate(ctx, value.ID, "old-hash", "replayed-hash", 2); !errors.Is(err, session.ErrRefreshTokenReused) {
		t.Fatalf("expected replay detection, got %v", err)
	}
	if _, err := store.Get(ctx, value.ID); !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("replayed session should be removed, got %v", err)
	}
}
