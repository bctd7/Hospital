package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
)

type AccessTokenIssuer interface {
	Issue(principal authn.Principal) (string, time.Time, error)
}

// Manager 编排 Refresh Session 的创建、轮换和注销。
// 它不关心 Redis 和 MySQL 的具体写法，只通过 Store、PrincipalStore 和 AccessTokenIssuer 协作。
type Manager struct {
	store      Store
	principals PrincipalStore
	issuer     AccessTokenIssuer
	versions   authn.AuthorizationVersionAdvancer
	refreshTTL time.Duration
	now        func() time.Time
}

// TokenPair 是一次登录或刷新后返回给客户端的完整凭证及其绝对过期时间。
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// NewManager 组装会话管理器；refreshTTL 控制一条登录会话的绝对生命周期。
func NewManager(
	store Store,
	principals PrincipalStore,
	issuer AccessTokenIssuer,
	versions authn.AuthorizationVersionAdvancer,
	refreshTTL time.Duration,
) (*Manager, error) {
	if store == nil || principals == nil || issuer == nil || versions == nil {
		return nil, errors.New("refresh session dependencies are required")
	}
	if refreshTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}
	return &Manager{
		store: store, principals: principals, issuer: issuer, versions: versions,
		refreshTTL: refreshTTL, now: time.Now,
	}, nil
}

// Start 在微信或短信等登录方式已经确认账号身份后，首次签发 Access Token 和 Refresh Token。
// 它只供 Identity 内部登录逻辑调用，不开放“传账号 ID 直接领 Token”的公共 RPC。
func (m *Manager) Start(ctx context.Context, accountID string) (TokenPair, error) {
	principal, err := m.principals.GetAuthorizationContext(ctx, accountID)
	if err != nil {
		return TokenPair{}, err
	}
	authPrincipal, err := activePrincipal(principal)
	if err != nil {
		return TokenPair{}, err
	}

	sessionID := uuid.NewString()
	familyID := uuid.NewString()
	rawRefresh, refreshHash, err := newRefreshToken(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	accessToken, accessExpiresAt, err := m.issuer.Issue(authPrincipal)
	if err != nil {
		return TokenPair{}, err
	}
	if _, err := m.versions.AdvanceAuthorizationVersion(ctx, authPrincipal.AccountID, authPrincipal.AuthorizationVersion); err != nil {
		return TokenPair{}, fmt.Errorf("advance authorization version: %w", err)
	}
	refreshExpiresAt := m.now().UTC().Add(m.refreshTTL)
	if err := m.store.Create(ctx, Session{
		ID: sessionID, FamilyID: familyID, AccountID: accountID, TokenHash: refreshHash,
		AuthorizationVersion: authPrincipal.AuthorizationVersion, ExpiresAt: refreshExpiresAt,
	}); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken: accessToken, RefreshToken: rawRefresh,
		AccessExpiresAt: accessExpiresAt, RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// Refresh 校验现有 Refresh Token，重新读取账号最新权限，并一次性轮换 Access Token 和 Refresh Token。
// 旧 Refresh Token 再次出现时会撤销当前会话，要求用户重新登录。
func (m *Manager) Refresh(ctx context.Context, rawRefresh string) (TokenPair, error) {
	sessionID, presentedHash, err := parseRefreshToken(rawRefresh)
	if err != nil {
		return TokenPair{}, err
	}
	current, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return TokenPair{}, err
	}
	if !tokenHashMatches(current.TokenHash, presentedHash) {
		// A token with a valid session id but an old secret is a replay attempt.
		_ = m.store.Revoke(ctx, sessionID, current.TokenHash)
		return TokenPair{}, ErrRefreshTokenReused
	}
	if !current.ExpiresAt.After(m.now().UTC()) {
		_ = m.store.Revoke(ctx, sessionID, current.TokenHash)
		return TokenPair{}, ErrSessionExpired
	}

	principal, err := m.principals.GetAuthorizationContext(ctx, current.AccountID)
	if err != nil {
		return TokenPair{}, err
	}
	authPrincipal, err := activePrincipal(principal)
	if err != nil {
		_ = m.store.Revoke(ctx, sessionID, current.TokenHash)
		return TokenPair{}, err
	}
	if current.AuthorizationVersion != authPrincipal.AuthorizationVersion {
		_ = m.store.Revoke(ctx, sessionID, current.TokenHash)
		return TokenPair{}, ErrAuthorizationChanged
	}
	if _, err := m.versions.AdvanceAuthorizationVersion(ctx, authPrincipal.AccountID, authPrincipal.AuthorizationVersion); err != nil {
		return TokenPair{}, fmt.Errorf("advance authorization version: %w", err)
	}

	replacement, replacementHash, err := newRefreshToken(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	accessToken, accessExpiresAt, err := m.issuer.Issue(authPrincipal)
	if err != nil {
		return TokenPair{}, err
	}
	if err := m.store.Rotate(ctx, sessionID, current.TokenHash, replacementHash, authPrincipal.AuthorizationVersion); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken: accessToken, RefreshToken: replacement,
		AccessExpiresAt: accessExpiresAt, RefreshExpiresAt: current.ExpiresAt,
	}, nil
}

// Revoke 校验客户端提交的 Refresh Token 后删除对应 Redis Session，用于退出登录。
func (m *Manager) Revoke(ctx context.Context, rawRefresh string) error {
	sessionID, refreshHash, err := parseRefreshToken(rawRefresh)
	if err != nil {
		return err
	}
	return m.store.Revoke(ctx, sessionID, refreshHash)
}

// activePrincipal 阻止空账号或已停用账号创建、刷新登录凭证。
func activePrincipal(principal authn.Principal) (authn.Principal, error) {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return authn.Principal{}, authn.ErrInactiveAccount
	}
	return principal, nil
}
