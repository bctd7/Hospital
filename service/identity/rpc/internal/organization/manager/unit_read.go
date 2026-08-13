package manager

import (
	"context"
	"fmt"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/organization"
)

// GetManagedUnit 读取一个可由管理端操作的组织单元；医院根节点保持只读。
func (m *UnitManager) GetManagedUnit(ctx context.Context, operator authn.Principal, unitID string) (organization.Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return organization.Unit{}, err
	}
	unitID, err := normalizedUUID(unitID, "unit_id")
	if err != nil {
		return organization.Unit{}, err
	}
	unit, err := m.store.GetUnit(ctx, unitID)
	if err != nil {
		return organization.Unit{}, err
	}
	if unit.Type == organization.UnitTypeHospital {
		return organization.Unit{}, fmt.Errorf("%w: hospital root is read-only", organization.ErrForbidden)
	}
	return unit, nil
}

// ListManagedUnits 校验管理权限和父子层级后，读取院区或科室列表。
func (m *UnitManager) ListManagedUnits(ctx context.Context, operator authn.Principal, filter organization.ListFilter) ([]organization.Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return nil, err
	}
	if filter.Type != organization.UnitTypeCampus && filter.Type != organization.UnitTypeDepartment {
		return nil, fmt.Errorf("%w: unit_type must be campus or department", organization.ErrInvalid)
	}
	if filter.ParentID == nil {
		return nil, fmt.Errorf("%w: parent_id is required", organization.ErrInvalid)
	}
	parentID, err := normalizedUUID(*filter.ParentID, "parent_id")
	if err != nil {
		return nil, err
	}
	filter.ParentID = &parentID
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unsupported status %q", organization.ErrInvalid, *filter.Status)
	}
	parent, err := m.store.GetUnit(ctx, parentID)
	if err != nil {
		return nil, err
	}
	requiredParentType, _ := filter.Type.RequiredParentType()
	if parent.Type != requiredParentType {
		return nil, fmt.Errorf("%w: %s requires a %s parent", organization.ErrInvalidHierarchy, filter.Type, requiredParentType)
	}
	return m.store.ListUnits(ctx, filter)
}
