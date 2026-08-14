// Package patient 提供患者独有的预约生命周期能力。
package patient

import "errors"

// Manager 是患者预约生命周期的业务入口。
// 它处理本人预约生命周期，以及本人正式检查报告的只读查询。
type Manager struct {
	projects ProjectReader
	bookings BookingStore
	reports  ReportStore
	cache    Cache
	flights  flightGroup
}

// NewManager 显式接收项目、预约和报告的窄端口，避免依赖一个职责过宽的 Store。
func NewManager(projects ProjectReader, bookings BookingStore, reports ReportStore, cache Cache) (*Manager, error) {
	if projects == nil {
		return nil, errors.New("patient project reader is required")
	}
	if bookings == nil {
		return nil, errors.New("patient booking store is required")
	}
	if reports == nil {
		return nil, errors.New("patient report store is required")
	}
	return &Manager{projects: projects, bookings: bookings, reports: reports, cache: cache}, nil
}
