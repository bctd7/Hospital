package shared

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
)

const defaultPageSize int64 = 20

func requirePatientView(principal authn.Principal) error {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return common.ErrForbidden
	}
	allowed := principal.AccountType == authn.AccountTypePatient ||
		(principal.AccountType == authn.AccountTypeStaff &&
			(principal.HasRole(authn.RoleDepartmentDoctor) || principal.HasRole(authn.RoleSuperAdmin)))
	if !allowed {
		return common.ErrForbidden
	}
	if _, err := uuid.Parse(strings.TrimSpace(principal.AccountID)); err != nil {
		return common.ErrForbidden
	}
	return nil
}

func requireStaffRead(operator authn.Principal) error {
	if err := commonauthz.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return fmt.Errorf("%w: %v", common.ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return common.ErrForbidden
	}
	return nil
}

func requireDepartmentScope(operator authn.Principal, departmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) || strings.TrimSpace(operator.DepartmentID) != departmentID {
		return common.ErrForbidden
	}
	return nil
}

func scopedDepartment(operator authn.Principal, requested string) (string, error) {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return optionalUUID(requested, "owner_department_id")
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) {
		return "", common.ErrForbidden
	}
	departmentID, err := requiredUUID(operator.DepartmentID, "operator department_id")
	if err != nil || (strings.TrimSpace(requested) != "" && strings.TrimSpace(requested) != departmentID) {
		return "", common.ErrForbidden
	}
	return departmentID, nil
}

func requiredUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", common.ErrInvalid, field)
	}
	return parsed.String(), nil
}

func optionalUUID(value, field string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return requiredUUID(value, field)
}

func normalizePage(page, pageSize int64) (int64, int64, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > 100 || page-1 > (1<<63-1)/pageSize {
		return 0, 0, 0, common.ErrInvalid
	}
	return page, pageSize, (page - 1) * pageSize, nil
}
