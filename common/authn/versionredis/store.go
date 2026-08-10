package versionredis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const DefaultPrefix = "identity:authorization-version:"

type Client interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
}

type Store struct {
	client Client
	prefix string
}

func NewStore(client Client, prefix string) (*Store, error) {
	if client == nil {
		return nil, errors.New("authorization version redis client is required")
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = DefaultPrefix
	}
	return &Store{client: client, prefix: prefix}, nil
}

func (s *Store) CurrentAuthorizationVersion(ctx context.Context, accountID string) (int64, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return 0, errors.New("authorization version account id is required")
	}
	version, err := s.client.Get(ctx, s.key(accountID)).Int64()
	if err != nil {
		return 0, fmt.Errorf("read authorization version: %w", err)
	}
	if version <= 0 {
		return 0, errors.New("stored authorization version must be positive")
	}
	return version, nil
}

func (s *Store) SetAuthorizationVersion(ctx context.Context, accountID string, version int64) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return errors.New("authorization version account id is required")
	}
	if version <= 0 {
		return errors.New("authorization version must be positive")
	}
	if err := s.client.Set(ctx, s.key(accountID), version, 0).Err(); err != nil {
		return fmt.Errorf("write authorization version: %w", err)
	}
	return nil
}

func (s *Store) key(accountID string) string {
	return s.prefix + accountID
}
