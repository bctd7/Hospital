package patient

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

func normalizeUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", common.ErrInvalid, field)
	}
	return parsed.String(), nil
}

func requirePatient(principal authn.Principal) error {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return common.ErrForbidden
	}
	allowed := principal.AccountType == authn.AccountTypePatient ||
		(principal.AccountType == authn.AccountTypeStaff &&
			(principal.HasRole(authn.RoleDepartmentDoctor) || principal.HasRole(authn.RoleSuperAdmin)))
	if !allowed {
		return common.ErrForbidden
	}
	if _, err := normalizeUUID(principal.AccountID, "patient account_id"); err != nil {
		return common.ErrForbidden
	}
	return nil
}
