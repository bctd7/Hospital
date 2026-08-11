package versionredis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

const DefaultPrefix = "identity:authorization-version:"

type Client interface {
	Get(ctx context.Context, key string) *redis.StringCmd

	Eval(
		ctx context.Context,
		script string,
		keys []string,
		args ...any,
	) *redis.Cmd
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

// authorizationVersionAdvancerScript 只允许权限版本向前推进。
const authorizationVersionAdvancerScript = `
local current = redis.call("GET", KEYS[1])

if current and tonumber(current) >= tonumber(ARGV[1]) then
	return 0
end

redis.call("SET", KEYS[1], ARGV[1])
return 1
`

func (s *Store) AdvanceAuthorizationVersion(
	ctx context.Context,
	accountID string,
	version int64,
) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return false, errors.New("authorization version account id is required")
	}
	if version <= 0 {
		return false, errors.New("authorization version must be positive")
	}
	result, err := s.client.Eval(
		ctx,
		authorizationVersionAdvancerScript,
		[]string{s.key(accountID)},
		version,
	).Int64()
	if err != nil {
		return false, fmt.Errorf(
			"advance authorization version: %w",
			err,
		)
	}
	switch result {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf(
			"unexpected advance authorization version result: %d",
			result,
		)
	}
}

func (s *Store) key(accountID string) string {
	return s.prefix + accountID
}
