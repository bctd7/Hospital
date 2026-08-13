// Package manager 编排组织目录读取和组织单元管理操作。
// 组织模型、稳定错误及 Store 端口由父包 organization 定义。
package manager

import "hospital/service/identity/rpc/internal/organization"

const (
	ActionUnitCreated  = "identity.organization.unit.created"
	ActionUnitUpdated  = "identity.organization.unit.updated"
	ActionUnitDisabled = "identity.organization.unit.disabled"
	ActionUnitEnabled  = "identity.organization.unit.enabled"
)

type CreateUnitCommand struct {
	Type        organization.UnitType
	ParentID    string
	Name        string
	OperationID string
	RequestID   string
}

type UpdateUnitCommand struct {
	UnitID          string
	Name            *string
	ParentID        *string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ChangeUnitStatusCommand struct {
	UnitID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type UnitManager struct {
	store organization.Store
}

func NewUnitManager(store organization.Store) *UnitManager {
	return &UnitManager{store: store}
}
