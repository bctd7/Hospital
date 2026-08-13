package authentication

import (
	"context"

	"hospital/service/identity/rpc/internal/session"
)

// Store 在短信验证成功后按手机号找到或创建账号。
type Store interface {
	FindOrCreateVerifiedPhoneAccount(ctx context.Context, fingerprint []byte, masked string) (string, error)
}

// SessionStarter 在认证成功后创建医院自己的 Access/Refresh 会话。
type SessionStarter interface {
	Start(ctx context.Context, accountID string) (session.TokenPair, error)
}
