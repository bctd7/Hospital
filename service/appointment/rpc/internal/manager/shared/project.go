package shared

import (
	"context"
	"errors"
	"fmt"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

// ListPatientProjects 只返回指定科室当前启用、可展示给患者的项目。
func (m *Manager) ListPatientProjects(ctx context.Context, patient authn.Principal, departmentID string, page, pageSize int64) (common.Page[common.ExaminationItem], error) {
	if err := requirePatientView(patient); err != nil {
		return common.Page[common.ExaminationItem]{}, err
	}
	departmentID, err := requiredUUID(departmentID, "department_id")
	if err != nil {
		return common.Page[common.ExaminationItem]{}, err
	}
	page, pageSize, offset, err := normalizePage(page, pageSize)
	if err != nil {
		return common.Page[common.ExaminationItem]{}, err
	}
	key := fmt.Sprintf("department:%s:g:%s:patient-projects:%d:%d", departmentID, m.generation(ctx, departmentID), page, pageSize)
	result, _, err := common.LoadCached(ctx, m.cache, &m.flights, key, common.HotReadCacheTTL, func() (common.Page[common.ExaminationItem], bool, error) {
		values, total, loadErr := m.store.ListItems(ctx, common.ProjectListFilter{OwnerDepartmentID: departmentID, Status: common.StatusActive, Offset: offset, Limit: pageSize})
		return common.Page[common.ExaminationItem]{Items: values, Page: page, PageSize: pageSize, Total: total}, loadErr == nil, loadErr
	})
	return result, err
}

// GetStaffProject 返回工作人员权限范围内的项目详情。
func (m *Manager) GetStaffProject(ctx context.Context, operator authn.Principal, itemID string) (common.ExaminationItem, error) {
	if err := requireStaffRead(operator); err != nil {
		return common.ExaminationItem{}, err
	}
	itemID, err := requiredUUID(itemID, "item_id")
	if err != nil {
		return common.ExaminationItem{}, err
	}
	item, err := m.store.GetItem(ctx, itemID)
	if err != nil {
		return common.ExaminationItem{}, err
	}
	if err := requireDepartmentScope(operator, item.OwnerDepartmentID); err != nil {
		return common.ExaminationItem{}, err
	}
	return item, nil
}

// GetProjectReference 返回 Guidance 等后端能力需要的项目公开引用。
// 项目名称和所属科室本来就是患者目录可见信息，因此工作人员可以跨科室读取；
// 这里不返回报告模板、周窗口或其他管理字段。
func (m *Manager) GetProjectReference(ctx context.Context, operator authn.Principal, itemID string) (common.ItemSummary, error) {
	if err := requireStaffRead(operator); err != nil {
		return common.ItemSummary{}, err
	}
	itemID, err := requiredUUID(itemID, "item_id")
	if err != nil {
		return common.ItemSummary{}, err
	}
	return m.store.GetItemSummary(ctx, itemID)
}

// ListStaffProjects 返回工作人员权限范围内的管理目录，可按状态筛选。
func (m *Manager) ListStaffProjects(ctx context.Context, operator authn.Principal, query ListProjectsQuery) (ListProjectsResult, error) {
	if err := requireStaffRead(operator); err != nil {
		return ListProjectsResult{}, err
	}
	departmentID, err := scopedDepartment(operator, query.OwnerDepartmentID)
	if err != nil {
		return ListProjectsResult{}, err
	}
	if query.Status != "" && !query.Status.Valid() {
		return ListProjectsResult{}, common.ErrInvalid
	}
	page, pageSize, offset, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return ListProjectsResult{}, err
	}
	items, total, err := m.store.ListItems(ctx, common.ProjectListFilter{OwnerDepartmentID: departmentID, Status: query.Status, Offset: offset, Limit: pageSize})
	if err != nil {
		return ListProjectsResult{}, err
	}
	return ListProjectsResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// ListPatientAvailableRooms 返回患者可自行选择的项目执行房间。
func (m *Manager) ListPatientAvailableRooms(ctx context.Context, patient authn.Principal, itemID string) ([]common.RoomItem, error) {
	item, err := m.patientItem(ctx, patient, itemID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:rooms:true", item.DepartmentID, m.generation(ctx, item.DepartmentID), item.ItemID)
	result, _, err := common.LoadCached(ctx, m.cache, &m.flights, key, common.QueryCacheTTL, func() ([]common.RoomItem, bool, error) {
		value, loadErr := m.store.ListItemRooms(ctx, item.ItemID, true)
		return value, true, loadErr
	})
	return result, err
}

// ListPatientProjectWindows 只返回患者侧可见的启用项目窗口。
func (m *Manager) ListPatientProjectWindows(ctx context.Context, patient authn.Principal, itemID string) ([]common.ItemWeeklyWindow, error) {
	item, err := m.patientItem(ctx, patient, itemID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:windows:true", item.DepartmentID, m.generation(ctx, item.DepartmentID), item.ItemID)
	result, _, err := common.LoadCached(ctx, m.cache, &m.flights, key, common.HotReadCacheTTL, func() ([]common.ItemWeeklyWindow, bool, error) {
		value, loadErr := m.store.ListItemWindows(ctx, item.ItemID, true)
		return value, true, loadErr
	})
	return result, err
}

func (m *Manager) patientItem(ctx context.Context, patient authn.Principal, itemID string) (common.ItemSummary, error) {
	if err := requirePatientView(patient); err != nil {
		return common.ItemSummary{}, err
	}
	itemID, err := requiredUUID(itemID, "item_id")
	if err != nil {
		return common.ItemSummary{}, err
	}
	value, found, err := common.LoadCached(ctx, m.cache, &m.flights, "item-summary:"+itemID, common.HotReadCacheTTL, func() (common.ItemSummary, bool, error) {
		item, loadErr := m.store.GetItemSummary(ctx, itemID)
		if errors.Is(loadErr, common.ErrNotFound) {
			return common.ItemSummary{}, false, nil
		}
		return item, loadErr == nil, loadErr
	})
	if err != nil {
		return common.ItemSummary{}, err
	}
	if !found || value.Status != common.StatusActive {
		return common.ItemSummary{}, common.ErrNotFound
	}
	return value, nil
}

func (m *Manager) generation(ctx context.Context, departmentID string) string {
	if m.cache == nil {
		return "0"
	}
	value, err := m.cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}
