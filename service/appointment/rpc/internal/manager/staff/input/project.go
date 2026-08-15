package input

import (
	"fmt"

	"hospital/service/appointment/rpc/internal/manager/common"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

// CreateProject 是创建检查项目时允许提交的字段。
type CreateProject struct {
	OwnerDepartmentID        string
	Name                     string
	Description              string
	EstimatedDurationMinutes int32
	OperationID              string
	RequestID                string
}

// UpdateProject 是修改检查项目基本信息时允许提交的字段。
// 指针字段用于区分“没有提交”和“提交为空值”。
type UpdateProject struct {
	ItemID                   string
	Name                     *string
	Description              *string
	EstimatedDurationMinutes *int32
	ExpectedVersion          int64
	OperationID              string
	RequestID                string
}

// ChangeProjectStatus 是启用或停用检查项目时允许提交的字段。
type ChangeProjectStatus struct {
	ItemID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

// SetItemWindow 是设置项目预约时间时允许提交的字段。
type SetItemWindow struct {
	WindowID          string
	ItemID            string
	Weekday           int32
	Session           common.Session
	StartTime         string
	BookingCutoffTime string
	EndTime           string
	ExpectedVersion   int64
	Operation
}

// NormalizeCreateProject 校验并规范化创建项目输入。
func NormalizeCreateProject(value CreateProject) (CreateProject, error) {
	var err error
	value.OwnerDepartmentID, err = staffsupport.NormalizeUUID(value.OwnerDepartmentID, "owner_department_id")
	if err != nil {
		return CreateProject{}, err
	}
	value.Name, err = staffsupport.NormalizeProjectName(value.Name)
	if err != nil {
		return CreateProject{}, err
	}
	value.Description, err = staffsupport.NormalizeProjectDescription(value.Description)
	if err != nil {
		return CreateProject{}, err
	}
	if err = validateEstimatedDuration(value.EstimatedDurationMinutes); err != nil {
		return CreateProject{}, err
	}
	value.OperationID, value.RequestID, err = staffsupport.NormalizeOperation(value.OperationID, value.RequestID)
	if err != nil {
		return CreateProject{}, err
	}
	return value, nil
}

// NormalizeUpdateProject 校验输入整体规则，并保留未提交的可选字段。
func NormalizeUpdateProject(value UpdateProject) (UpdateProject, error) {
	var err error
	value.ItemID, err = staffsupport.NormalizeUUID(value.ItemID, "item_id")
	if err != nil {
		return UpdateProject{}, err
	}
	if value.ExpectedVersion <= 0 {
		return UpdateProject{}, fmt.Errorf("%w: expected_version must be positive", common.ErrInvalid)
	}
	if value.Name == nil && value.Description == nil && value.EstimatedDurationMinutes == nil {
		return UpdateProject{}, fmt.Errorf("%w: name, description or estimated_duration_minutes is required", common.ErrInvalid)
	}
	if value.Name != nil {
		name, normalizeErr := staffsupport.NormalizeProjectName(*value.Name)
		if normalizeErr != nil {
			return UpdateProject{}, normalizeErr
		}
		value.Name = &name
	}
	if value.Description != nil {
		description, normalizeErr := staffsupport.NormalizeProjectDescription(*value.Description)
		if normalizeErr != nil {
			return UpdateProject{}, normalizeErr
		}
		value.Description = &description
	}
	if value.EstimatedDurationMinutes != nil {
		if durationErr := validateEstimatedDuration(*value.EstimatedDurationMinutes); durationErr != nil {
			return UpdateProject{}, durationErr
		}
	}
	value.OperationID, value.RequestID, err = staffsupport.NormalizeOperation(value.OperationID, value.RequestID)
	if err != nil {
		return UpdateProject{}, err
	}
	return value, nil
}

func validateEstimatedDuration(value int32) error {
	if value < 5 || value > 480 || value%5 != 0 {
		return fmt.Errorf("%w: estimated_duration_minutes must be between 5 and 480 and a multiple of 5", common.ErrInvalid)
	}
	return nil
}

// NormalizeChangeProjectStatus 校验项目状态变更输入。
func NormalizeChangeProjectStatus(value ChangeProjectStatus) (ChangeProjectStatus, error) {
	var err error
	value.ItemID, err = staffsupport.NormalizeUUID(value.ItemID, "item_id")
	if err != nil {
		return ChangeProjectStatus{}, err
	}
	if value.ExpectedVersion <= 0 {
		return ChangeProjectStatus{}, fmt.Errorf("%w: expected_version must be positive", common.ErrInvalid)
	}
	value.OperationID, value.RequestID, err = staffsupport.NormalizeOperation(value.OperationID, value.RequestID)
	if err != nil {
		return ChangeProjectStatus{}, err
	}
	return value, nil
}
