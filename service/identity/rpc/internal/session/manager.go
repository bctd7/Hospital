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

type Manager struct {
	store      Store
	principals PrincipalStore
	issuer     AccessTokenIssuer
	refreshTTL time.Duration
	now        func() time.Time
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

func NewManager(store Store, principals PrincipalStore, issuer AccessTokenIssuer, refreshTTL time.Duration) (*Manager, error) {
	if store == nil || principals == nil || issuer == nil {
		return nil, errors.New("refresh session dependencies are required")
	}
	if refreshTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}
	return &Manager{store: store, principals: principals, issuer: issuer, refreshTTL: refreshTTL, now: time.Now}, nil
}

// Start creates a server-side refresh session after a login provider has
// authenticated the account. It is intentionally not exposed as a public RPC.
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

func (m *Manager) Revoke(ctx context.Context, rawRefresh string) error {
	sessionID, refreshHash, err := parseRefreshToken(rawRefresh)
	if err != nil {
		return err
	}
	return m.store.Revoke(ctx, sessionID, refreshHash)
}

func activePrincipal(principal authn.Principal) (authn.Principal, error) {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return authn.Principal{}, authn.ErrInactiveAccount
	}
	return principal, nil
}
