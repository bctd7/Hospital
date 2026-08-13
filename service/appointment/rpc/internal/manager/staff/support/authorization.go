// Package support 提供 staff.Manager 使用的纯辅助能力。
// 本包不编排业务、不持有 Store，也不定义新的 Manager。
package support

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
)

// RequirePermission 校验工作人员权限和操作者身份格式。
func RequirePermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return fmt.Errorf("%w: %v", common.ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return fmt.Errorf("%w: invalid operator identity", common.ErrForbidden)
	}
	return nil
}

// RequireDepartmentScope 限制科室工作人员只能操作本人所属科室的数据。
func RequireDepartmentScope(operator authn.Principal, departmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) {
		return common.ErrForbidden
	}
	own, err := NormalizeUUID(operator.DepartmentID, "operator department_id")
	if err != nil || own != departmentID {
		return common.ErrForbidden
	}
	return nil
}

// ScopedDepartment 返回当前工作人员允许访问的科室编号。
func ScopedDepartment(operator authn.Principal, requested string) (string, error) {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return NormalizeUUID(requested, "department_id")
	}
	own, err := NormalizeUUID(operator.DepartmentID, "operator department_id")
	if err != nil {
		return "", common.ErrForbidden
	}
	if requested != "" {
		requested, err = NormalizeUUID(requested, "department_id")
		if err != nil || requested != own {
			return "", common.ErrForbidden
		}
	}
	return own, nil
}
