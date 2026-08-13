// Package patient 提供患者独有的预约生命周期能力。
package patient

import "errors"

// Manager 是患者预约生命周期的业务入口。
// 它只处理预约选项、创建预约、查询本人预约和取消本人预约。
type Manager struct {
	projects ProjectReader
	bookings BookingStore
	cache    Cache
	flights  flightGroup
}

// NewManager 显式接收项目最小读取端口和预约存储端口，避免依赖一个职责过宽的 Store。
func NewManager(projects ProjectReader, bookings BookingStore, cache Cache) (*Manager, error) {
	if projects == nil {
		return nil, errors.New("patient project reader is required")
	}
	if bookings == nil {
		return nil, errors.New("patient booking store is required")
	}
	return &Manager{projects: projects, bookings: bookings, cache: cache}, nil
}
