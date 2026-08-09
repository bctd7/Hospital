package session

import (
	"context"
	"errors"
	"time"

	"hospital/common/authn"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrSessionNotFound     = errors.New("refresh session not found")
	ErrSessionExpired      = errors.New("refresh session expired")
	ErrRefreshTokenReused  = errors.New("refresh token was already used")
)

// Session 是一条保存在服务端的刷新会话。
// 客户端只持有原始 Refresh Token；Redis 只保存 TokenHash，避免 Redis 数据泄漏后凭证被直接使用。
type Session struct {
	// ID 同时是 Redis Key 的一部分，也是 Refresh Token 中可公开的会话定位符。
	ID string
	// FamilyID 标识同一次登录产生的 Token 链，预留给多次轮换和重放追踪使用。
	FamilyID string
	// AccountID 指向 MySQL 中的账号；刷新时需要用它重新读取最新权限。
	AccountID string
	// TokenHash 是完整 Refresh Token 的 SHA-256 哈希，不保存 Token 原文。
	TokenHash string
	// AuthorizationVersion 记录创建或刷新会话时的授权版本，便于后续失效策略扩展。
	AuthorizationVersion int64
	// ExpiresAt 是绝对过期时间；Token 轮换不会延长它。
	ExpiresAt time.Time
}

// Store 是 Refresh Session 的存储端口。
// Manager 只依赖这个接口，不直接依赖 Redis；生产环境由 repository.RedisSessionStore 实现，单元测试可使用内存实现。
type Store interface {
	// Create 在首次登录成功后保存一条新的刷新会话。
	Create(ctx context.Context, session Session) error
	// Get 根据 Refresh Token 中的 sessionID 读取当前会话。
	Get(ctx context.Context, sessionID string) (Session, error)
	// Rotate 仅在 expectedHash 仍是当前哈希时替换为 replacementHash，用于保证一次性轮换和并发安全。
	Rotate(ctx context.Context, sessionID, expectedHash, replacementHash string, authorizationVersion int64) error
	// Revoke 仅在 Token 哈希匹配时删除会话；会话不存在时应保持幂等。
	Revoke(ctx context.Context, sessionID, expectedHash string) error
}

// PrincipalStore 提供刷新 Token 时所需的最新账号授权事实。
// 当前由 MySQLStore 实现，确保新 Access Token 不继续携带过期的角色、科室或账号状态。
type PrincipalStore interface {
	GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error)
}
