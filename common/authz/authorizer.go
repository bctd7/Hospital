package authz

import (
	"errors"
	"fmt"
	"strings"

	"hospital/common/authn"
)

var (
	ErrUnauthenticated  = errors.New("account is not authenticated")
	ErrPermissionDenied = errors.New("permission denied")
	ErrScopeDenied      = errors.New("resource scope denied")
)

type AccessMode string

const (
	AccessRead  AccessMode = "read"
	AccessWrite AccessMode = "write"
)

type Resource struct {
	Type         string
	ID           string
	DepartmentID string
}

func RequirePermission(principal authn.Principal, permission string) error {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return ErrUnauthenticated
	}
	if !principal.HasPermission(permission) {
		return fmt.Errorf("%w: %s", ErrPermissionDenied, permission)
	}
	return nil
}

func AuthorizeDepartment(principal authn.Principal, permission string, mode AccessMode, resource Resource) error {
	if err := RequirePermission(principal, permission); err != nil {
		return err
	}
	if principal.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !principal.HasRole(authn.RoleDepartmentDoctor) {
		return ErrScopeDenied
	}
	if resource.DepartmentID == "" {
		return fmt.Errorf("%w: resource department is required", ErrScopeDenied)
	}
	if principal.DepartmentID == resource.DepartmentID {
		return nil
	}
	if mode == AccessRead && isReadPermission(permission) {
		return nil
	}
	return fmt.Errorf("%w: account department %q cannot access department %q", ErrScopeDenied, principal.DepartmentID, resource.DepartmentID)
}

func isReadPermission(permission string) bool {
	return strings.HasSuffix(permission, ".read")
}
