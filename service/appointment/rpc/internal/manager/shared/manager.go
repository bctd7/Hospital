// Package shared 提供患者端与工作人员端共同使用的业务能力。
// 当前主要承载项目、房间和窗口读取；某一角色独有的业务动作不放在这里。
package shared

import (
	"errors"

	"hospital/service/appointment/rpc/internal/manager/common"
)

// Manager 是双方共用业务能力的统一入口。
type Manager struct {
	store   Store
	cache   common.Cache
	flights common.FlightGroup
}

// NewManager 组装双方共用业务所需的持久化和缓存能力。
func NewManager(store Store, cache common.Cache) (*Manager, error) {
	if store == nil {
		return nil, errors.New("shared appointment store is required")
	}
	return &Manager{store: store, cache: cache}, nil
}
