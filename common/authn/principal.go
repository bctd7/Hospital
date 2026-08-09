package authn

import (
	"context"
	"errors"
)

const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"

	AccountTypePatient = "patient"
	AccountTypeStaff   = "staff"
	AccountTypeService = "service"

	RoleSuperAdmin       = "super_admin"
	RoleDepartmentDoctor = "department_doctor"
)

var ErrPrincipalMissing = errors.New("authentication principal is missing")

type Principal struct {
	AccountID            string
	AccountType          string
	Status               string
	Roles                []string
	DepartmentID         string
	Permissions          []string
	AuthorizationVersion int64
}

func (p Principal) HasRole(role string) bool {
	for _, current := range p.Roles {
		if current == role {
			return true
		}
	}
	return false
}

func (p Principal) HasPermission(permission string) bool {
	for _, current := range p.Permissions {
		if current == permission || current == "*" {
			return true
		}
	}
	return false
}

type principalContextKey struct{}

func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	if !ok || principal.AccountID == "" {
		return Principal{}, ErrPrincipalMissing
	}
	return principal, nil
}
