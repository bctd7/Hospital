package precedence

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

func requirePermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return ErrForbidden
	}
	return nil
}

func requireRead(operator authn.Principal) error {
	return requirePermission(operator, contractauthz.PermissionRuleRead)
}

func requireEdit(operator authn.Principal) error {
	return requirePermission(operator, contractauthz.PermissionRuleEdit)
}

func requireDepartmentScope(operator authn.Principal, departmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) || strings.TrimSpace(operator.DepartmentID) != departmentID {
		return ErrForbidden
	}
	return nil
}

func normalizeUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}

func normalizeRequiredText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return "", ErrInvalid
	}
	return value, nil
}

func normalizeOptionalText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return "", ErrInvalid
	}
	return value, nil
}

func normalizeDirection(value Direction) (Direction, error) {
	if value == "" {
		return DirectionAll, nil
	}
	if !value.Valid() {
		return "", ErrInvalid
	}
	return value, nil
}
