// Package session 管理 Access/Refresh 会话的创建、轮换、刷新和注销生命周期。
package session

import (
	"errors"
	"time"
)

var (
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrSessionNotFound      = errors.New("refresh session not found")
	ErrSessionExpired       = errors.New("refresh session expired")
	ErrRefreshTokenReused   = errors.New("refresh token was already used")
	ErrAuthorizationChanged = errors.New("authorization changed; login is required")
)

// Session 是一条保存在服务端的刷新会话；Redis 只保存 TokenHash，不保存可直接使用的 Token 原文。
type Session struct {
	ID                   string
	FamilyID             string
	AccountID            string
	TokenHash            string
	AuthorizationVersion int64
	ExpiresAt            time.Time
}

// TokenPair 是一次登录或刷新后返回给客户端的完整凭证及其绝对过期时间。
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}
