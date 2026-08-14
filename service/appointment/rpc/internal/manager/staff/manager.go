// Package staff 提供管理员和授权科室工作人员独有的管理及现场操作能力。
package staff

import (
	"context"
	"errors"

	"hospital/common/authn"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

// Manager 是管理员和授权科室工作人员的唯一 Appointment 业务入口。
// 根层按项目、房间、房间项目关系、预约、检查执行和报告拆文件，不形成并列 Manager。
type Manager struct {
	projectWrites ProjectWriteStore
	projects      ProjectStore
	rooms         RoomStore
	bookings      BookingStore
	reports       ReportStore
	cache         staffsupport.Cache
	flights       staffsupport.FlightGroup
}

// mutate 将 Manager 持有的事务端口交给 support 的幂等写模板。
// 业务状态变更仍由各业务文件传入的 apply 函数决定。
func (m *Manager) mutate(ctx context.Context, store staffsupport.OperationStore, operator authn.Principal, meta staffinput.Operation, resourceType, resourceID, action string, payload any, departmentID func() string, apply func(ConfigurationTxStore) (any, string, error), replay func([]byte) error) error {
	return staffsupport.Mutate(ctx, store, operator, staffsupport.OperationMeta{
		OperationID: meta.OperationID,
		RequestID:   meta.RequestID,
	}, resourceType, resourceID, action, payload, departmentID, apply, replay)
}

func (m *Manager) generation(ctx context.Context, departmentID string) string {
	return staffsupport.Generation(ctx, m.cache, departmentID)
}

func (m *Manager) getItemSummary(ctx context.Context, itemID string) (ItemSummary, error) {
	return staffsupport.GetItemSummary(ctx, m.projects, m.cache, &m.flights, itemID)
}

func (m *Manager) invalidate(ctx context.Context, departmentID string, keys ...string) {
	staffsupport.Invalidate(ctx, m.cache, departmentID, keys...)
}

// InvalidateItem 在项目事务提交后使项目摘要和科室范围缓存切换到新版本。
func (m *Manager) InvalidateItem(ctx context.Context, departmentID, itemID string) {
	m.invalidate(ctx, departmentID, "item-summary:"+itemID)
}

// NewManager 组装工作人员写操作、房间开放时间配置和预约处理所需的最小持久化端口。
func NewManager(projects ProjectStore, rooms RoomStore, bookingStore BookingStore, reportStore ReportStore, cache staffsupport.Cache) (*Manager, error) {
	if projects == nil {
		return nil, errors.New("appointment project store is required")
	}
	if rooms == nil {
		return nil, errors.New("appointment room store is required")
	}
	if bookingStore == nil {
		return nil, errors.New("appointment booking store is required")
	}
	if reportStore == nil {
		return nil, errors.New("appointment report store is required")
	}
	return &Manager{projectWrites: projects, projects: projects, rooms: rooms, bookings: bookingStore, reports: reportStore, cache: cache}, nil
}
