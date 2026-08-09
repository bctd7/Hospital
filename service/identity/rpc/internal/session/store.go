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

type Session struct {
	ID                   string
	FamilyID             string
	AccountID            string
	TokenHash            string
	AuthorizationVersion int64
	ExpiresAt            time.Time
}

type Store interface {
	Create(ctx context.Context, session Session) error
	Get(ctx context.Context, sessionID string) (Session, error)
	Rotate(ctx context.Context, sessionID, expectedHash, replacementHash string, authorizationVersion int64) error
	Revoke(ctx context.Context, sessionID, expectedHash string) error
}

type PrincipalStore interface {
	GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error)
}
