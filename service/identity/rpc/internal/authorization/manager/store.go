package manager

import (
	"context"
	"errors"

	"hospital/common/authn"
)

var (
	ErrNotFound  = errors.New("identity resource not found")
	ErrInvalid   = errors.New("invalid identity authorization request")
	ErrForbidden = errors.New("identity authorization operation is forbidden")
)

type Store interface {
	GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error)
}
