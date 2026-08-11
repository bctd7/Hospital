package authn

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrAuthorizationVersionUnavailable = errors.New("authorization version is unavailable")
	ErrAuthorizationVersionStale       = errors.New("access token authorization version is stale")
)

type AuthorizationVersionReader interface {
	CurrentAuthorizationVersion(ctx context.Context, accountID string) (int64, error)
}

type AuthorizationVersionWriter interface {
	SetAuthorizationVersion(ctx context.Context, accountID string, version int64) error
}

type PrincipalValidator interface {
	ValidatePrincipal(ctx context.Context, principal Principal) error
}

type AuthorizationVersionValidator struct {
	reader AuthorizationVersionReader
}

func NewAuthorizationVersionValidator(reader AuthorizationVersionReader) (*AuthorizationVersionValidator, error) {
	if reader == nil {
		return nil, errors.New("authorization version reader is required")
	}
	return &AuthorizationVersionValidator{reader: reader}, nil
}

func (v *AuthorizationVersionValidator) ValidatePrincipal(ctx context.Context, principal Principal) error {
	if principal.AccountID == "" || principal.AuthorizationVersion <= 0 {
		return ErrAuthorizationVersionStale
	}
	current, err := v.reader.CurrentAuthorizationVersion(ctx, principal.AccountID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthorizationVersionUnavailable, err)
	}
	if current <= 0 {
		return ErrAuthorizationVersionUnavailable
	}
	if current != principal.AuthorizationVersion {
		return ErrAuthorizationVersionStale
	}
	return nil
}
