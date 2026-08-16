package manager

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

func requirePermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return fmt.Errorf("%w: %v", precedence.ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return precedence.ErrForbidden
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
		return precedence.ErrForbidden
	}
	return nil
}

func normalizeUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", precedence.ErrInvalid
	}
	return parsed.String(), nil
}

func normalizeRequiredText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return "", precedence.ErrInvalid
	}
	return value, nil
}

func normalizeOptionalText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return "", precedence.ErrInvalid
	}
	return value, nil
}

func normalizeDirection(value precedence.Direction) (precedence.Direction, error) {
	if value == "" {
		return precedence.DirectionAll, nil
	}
	if !value.Valid() {
		return "", precedence.ErrInvalid
	}
	return value, nil
}
