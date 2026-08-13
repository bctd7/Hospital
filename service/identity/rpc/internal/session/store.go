package session

import (
	"context"

	"hospital/common/authn"
)

// Store 是 Refresh Session 的存储端口。
// Manager 只依赖这个接口，不直接依赖 Redis；生产环境由 repository/redisstore.SessionStore 实现，单元测试可使用内存实现。
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
	GetPrincipal(ctx context.Context, accountID string) (authn.Principal, error)
}
