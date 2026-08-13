// Package context 负责读取有效 Principal，并校验读取其他账号授权上下文所需的权限。
// 本包只读，不修改角色、权限、账号状态或科室范围。
package context

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
