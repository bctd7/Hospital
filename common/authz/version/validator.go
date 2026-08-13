// Package version validates token authorization versions and defines the
// monotonic version store boundary shared by Identity and backend services.
package version

import (
	"context"
	"errors"
	"fmt"

	"hospital/common/authn"
)

var (
	ErrAuthorizationVersionUnavailable = errors.New("authorization version is unavailable")
	ErrAuthorizationVersionStale       = errors.New("access token authorization version is stale")
)

type AuthorizationVersionReader interface {
	CurrentAuthorizationVersion(ctx context.Context, accountID string) (int64, error)
}

type Validator struct {
	reader AuthorizationVersionReader
}

func NewValidator(reader AuthorizationVersionReader) (*Validator, error) {
	if reader == nil {
		return nil, errors.New("authorization version reader is required")
	}
	return &Validator{reader: reader}, nil
}

func (v *Validator) ValidatePrincipal(ctx context.Context, principal authn.Principal) error {
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

type AuthorizationVersionAdvancer interface {
	AdvanceAuthorizationVersion(
		ctx context.Context,
		accountID string,
		version int64,
	) (updated bool, err error)
}
