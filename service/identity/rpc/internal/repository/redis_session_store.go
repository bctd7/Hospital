package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/service/identity/rpc/internal/session"
)

const defaultRefreshSessionPrefix = "identity:refresh:"

var rotateRefreshSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 0
end
if redis.call('HGET', KEYS[1], 'token_hash') ~= ARGV[1] then
  redis.call('DEL', KEYS[1])
  return -1
end
redis.call('HSET', KEYS[1],
  'token_hash', ARGV[2],
  'authorization_version', ARGV[3])
return 1
`)

var revokeRefreshSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 0
end
if redis.call('HGET', KEYS[1], 'token_hash') ~= ARGV[1] then
  return -1
end
redis.call('DEL', KEYS[1])
return 1
`)

type RedisSessionStore struct {
	client *redis.Client
	prefix string
}

// NewRedisSessionStore 创建 Store 的 Redis 实现；prefix 用于隔离不同环境或不同类型的 Key。
func NewRedisSessionStore(client *redis.Client, prefix string) (*RedisSessionStore, error) {
	if client == nil {
		return nil, errors.New("identity redis client is required")
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultRefreshSessionPrefix
	}
	return &RedisSessionStore{client: client, prefix: prefix}, nil
}

// Create 将首次登录产生的 Session 写为 Redis Hash，并设置绝对过期时间。
func (s *RedisSessionStore) Create(ctx context.Context, value session.Session) error {
	ttl := time.Until(value.ExpiresAt)
	if ttl <= 0 {
		return session.ErrSessionExpired
	}
	key := s.key(value.ID)
	_, err := s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, map[string]any{
			"family_id":             value.FamilyID,
			"account_id":            value.AccountID,
			"token_hash":            value.TokenHash,
			"authorization_version": value.AuthorizationVersion,
			"expires_at_unix":       value.ExpiresAt.Unix(),
		})
		pipe.ExpireAt(ctx, key, value.ExpiresAt)
		return nil
	})
	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

// Get 读取 Redis Hash，并还原为领域层使用的 session.Session。
func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (session.Session, error) {
	values, err := s.client.HGetAll(ctx, s.key(sessionID)).Result()
	if err != nil {
		return session.Session{}, fmt.Errorf("read refresh session: %w", err)
	}
	if len(values) == 0 {
		return session.Session{}, session.ErrSessionNotFound
	}
	version, err := strconv.ParseInt(values["authorization_version"], 10, 64)
	if err != nil {
		return session.Session{}, fmt.Errorf("decode refresh session authorization version: %w", err)
	}
	expiresAtUnix, err := strconv.ParseInt(values["expires_at_unix"], 10, 64)
	if err != nil {
		return session.Session{}, fmt.Errorf("decode refresh session expiry: %w", err)
	}
	return session.Session{
		ID: sessionID, FamilyID: values["family_id"], AccountID: values["account_id"],
		TokenHash: values["token_hash"], AuthorizationVersion: version,
		ExpiresAt: time.Unix(expiresAtUnix, 0).UTC(),
	}, nil
}

// Rotate 通过 Lua 在 Redis 内原子完成“比较旧哈希并写入新哈希”。
// 若哈希不一致，说明可能发生并发刷新或旧 Token 重放，脚本会删除 Session。
func (s *RedisSessionStore) Rotate(
	ctx context.Context,
	sessionID, expectedHash, replacementHash string,
	authorizationVersion int64,
) error {
	result, err := rotateRefreshSessionScript.Run(
		ctx, s.client, []string{s.key(sessionID)}, expectedHash, replacementHash, authorizationVersion,
	).Int64()
	if err != nil {
		return fmt.Errorf("rotate refresh session: %w", err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return session.ErrSessionNotFound
	case -1:
		return session.ErrRefreshTokenReused
	default:
		return fmt.Errorf("rotate refresh session: unexpected result %d", result)
	}
}

// Revoke 只删除与 expectedHash 匹配的 Session，避免仅知道 sessionID 就能让其他用户下线。
func (s *RedisSessionStore) Revoke(ctx context.Context, sessionID, expectedHash string) error {
	result, err := revokeRefreshSessionScript.Run(
		ctx, s.client, []string{s.key(sessionID)}, expectedHash,
	).Int64()
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	switch result {
	case 1, 0:
		return nil
	case -1:
		return session.ErrInvalidRefreshToken
	default:
		return fmt.Errorf("revoke refresh session: unexpected result %d", result)
	}
}

// key 统一生成 Redis Key，当前格式为 identity:refresh:{sessionID}。
func (s *RedisSessionStore) key(sessionID string) string {
	return s.prefix + sessionID
}
